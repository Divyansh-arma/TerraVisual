package drift

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"terra-parser/parser"
)

// DriftStatus constants
const (
	StatusInSync         = "IN_SYNC"
	StatusModified       = "MODIFIED"
	StatusMissingInState = "MISSING_IN_STATE"
	StatusMissingInCode  = "MISSING_IN_CODE"
)

// DetectDrift compares the stateGraph against the codeGraph using two-pass reconciliation
// (exact ID match + fuzzy attribute match) and produces granular attribute diffs.
func DetectDrift(stateGraph *parser.GraphResponse, codeGraph *parser.GraphResponse) *parser.GraphResponse {
	if stateGraph == nil {
		stateGraph = &parser.GraphResponse{}
	}
	if codeGraph == nil {
		codeGraph = &parser.GraphResponse{}
	}

	stateNodeMap := make(map[string]parser.Node)
	for _, n := range stateGraph.Nodes {
		stateNodeMap[n.ID] = n
	}

	codeNodeMap := make(map[string]parser.Node)
	for _, n := range codeGraph.Nodes {
		codeNodeMap[n.ID] = n
	}

	// Index dependencies per source resource
	stateDeps := make(map[string][]string)
	for _, e := range stateGraph.Edges {
		stateDeps[e.Source] = append(stateDeps[e.Source], e.Target)
	}

	codeDeps := make(map[string][]string)
	for _, e := range codeGraph.Edges {
		codeDeps[e.Source] = append(codeDeps[e.Source], e.Target)
	}

	// Merge all edges from both graphs
	edgeMap := make(map[string]parser.Edge)
	for _, e := range stateGraph.Edges {
		edgeMap[e.ID] = e
	}
	for _, e := range codeGraph.Edges {
		edgeMap[e.ID] = e
	}

	matchedStateIDs := make(map[string]bool)
	matchedCodeIDs := make(map[string]bool)

	type MatchedPair struct {
		StateNode parser.Node
		CodeNode  parser.Node
	}
	var matchedPairs []MatchedPair

	// Pass 1: Exact ID matching
	for id, codeNode := range codeNodeMap {
		if stateNode, ok := stateNodeMap[id]; ok {
			matchedPairs = append(matchedPairs, MatchedPair{
				StateNode: stateNode,
				CodeNode:  codeNode,
			})
			matchedStateIDs[id] = true
			matchedCodeIDs[id] = true
		}
	}

	// Pass 2: Fuzzy / Attribute Matching for remaining unmatched nodes
	for codeID, codeNode := range codeNodeMap {
		if matchedCodeIDs[codeID] {
			continue
		}

		bestMatchStateID := ""
		for stateID, stateNode := range stateNodeMap {
			if matchedStateIDs[stateID] {
				continue
			}

			if isFuzzyMatch(codeNode, stateNode) {
				bestMatchStateID = stateID
				break
			}
		}

		if bestMatchStateID != "" {
			matchedPairs = append(matchedPairs, MatchedPair{
				StateNode: stateNodeMap[bestMatchStateID],
				CodeNode:  codeNode,
			})
			matchedStateIDs[bestMatchStateID] = true
			matchedCodeIDs[codeID] = true
		}
	}

	mergedNodes := make([]parser.Node, 0, len(stateNodeMap)+len(codeNodeMap))

	// 1. Process all matched pairs (IN_SYNC or MODIFIED)
	for _, pair := range matchedPairs {
		stateNode := pair.StateNode
		codeNode := pair.CodeNode

		diffs := computeAttributeDiffs(stateNode, codeNode)
		depsDiff := !compareDeps(stateDeps[stateNode.ID], codeDeps[codeNode.ID])

		driftStatus := StatusInSync
		if len(diffs) > 0 || depsDiff {
			driftStatus = StatusModified
		}

		// Merge attributes: copy state attributes, overlay code attributes
		mergedAttrs := make(map[string]interface{})
		for k, v := range stateNode.Data.Attributes {
			mergedAttrs[k] = v
		}
		for k, v := range codeNode.Data.Attributes {
			mergedAttrs[k] = v
		}

		parentID := codeNode.ParentID
		if parentID == "" {
			parentID = stateNode.ParentID
		}

		mergedNode := codeNode
		if mergedNode.ID == "" {
			mergedNode = stateNode
		}
		mergedNode.ParentID = parentID
		mergedNode.Data.Attributes = mergedAttrs
		mergedNode.Data.DriftStatus = driftStatus
		mergedNode.Data.DriftDiffs = diffs

		mergedNodes = append(mergedNodes, mergedNode)
	}

	// 2. Process unmatched code nodes -> MISSING_IN_STATE (Declared in code, unapplied)
	for codeID, codeNode := range codeNodeMap {
		if !matchedCodeIDs[codeID] {
			node := codeNode
			node.Data.DriftStatus = StatusMissingInState
			mergedNodes = append(mergedNodes, node)
		}
	}

	// 3. Process unmatched state nodes -> MISSING_IN_CODE (Present in state, unmanaged in code)
	for stateID, stateNode := range stateNodeMap {
		if !matchedStateIDs[stateID] {
			node := stateNode
			node.Data.DriftStatus = StatusMissingInCode
			mergedNodes = append(mergedNodes, node)
		}
	}

	mergedEdges := make([]parser.Edge, 0, len(edgeMap))
	for _, e := range edgeMap {
		mergedEdges = append(mergedEdges, e)
	}

	// Calculate IsContainer for all merged nodes
	parentSet := make(map[string]bool)
	for _, n := range mergedNodes {
		if n.ParentID != "" {
			parentSet[n.ParentID] = true
		}
	}

	for i := range mergedNodes {
		id := mergedNodes[i].ID
		isParentContainer := parentSet[id]
		if mergedNodes[i].Data.ResourceType == "module" || mergedNodes[i].Data.ResourceType == "aws_vpc" || mergedNodes[i].Data.ResourceType == "azurerm_virtual_network" || mergedNodes[i].Data.ResourceType == "google_compute_network" {
			mergedNodes[i].Data.IsContainer = isParentContainer
		}
	}

	// Sort nodes with parents before children, and sort edges deterministically
	parser.SortNodesParentsFirst(mergedNodes)

	sort.Slice(mergedEdges, func(i, j int) bool {
		return mergedEdges[i].ID < mergedEdges[j].ID
	})

	return &parser.GraphResponse{
		Nodes: mergedNodes,
		Edges: mergedEdges,
	}
}

// isFuzzyMatch tests if two resources match by resource type and primary architectural identifiers.
func isFuzzyMatch(a, b parser.Node) bool {
	// Must share the same resource type or general cloud provider family
	if a.Data.ResourceType != b.Data.ResourceType {
		return false
	}

	rt := strings.ToLower(a.Data.ResourceType)
	attrsA := a.Data.Attributes
	attrsB := b.Data.Attributes
	if attrsA == nil || attrsB == nil {
		return false
	}

	// Subnets & VPCs: CIDR match
	if strings.Contains(rt, "subnet") || strings.Contains(rt, "vpc") {
		cidrA := getAttrString(attrsA, "cidr_block", "cidr")
		cidrB := getAttrString(attrsB, "cidr_block", "cidr")
		if cidrA != "" && cidrA == cidrB {
			return true
		}
	}

	// S3 Buckets: bucket name match
	if strings.Contains(rt, "s3") || strings.Contains(rt, "bucket") {
		bucketA := getAttrString(attrsA, "bucket", "bucket_name")
		bucketB := getAttrString(attrsB, "bucket", "bucket_name")
		if bucketA != "" && bucketA == bucketB {
			return true
		}
	}

	// DynamoDB Tables: table name match
	if strings.Contains(rt, "dynamo") || strings.Contains(rt, "table") {
		tableA := getAttrString(attrsA, "name", "table_name")
		tableB := getAttrString(attrsB, "name", "table_name")
		if tableA != "" && tableA == tableB {
			return true
		}
	}

	// EKS Clusters: cluster name match
	if strings.Contains(rt, "eks") || strings.Contains(rt, "cluster") {
		nameA := getAttrString(attrsA, "name", "cluster_name")
		nameB := getAttrString(attrsB, "name", "cluster_name")
		if nameA != "" && nameA == nameB {
			return true
		}
	}

	// Fallback Name Tag Match
	nameTagA := getNameTag(attrsA)
	nameTagB := getNameTag(attrsB)
	if nameTagA != "" && nameTagA == nameTagB {
		return true
	}

	return false
}

// computeAttributeDiffs calculates attribute divergence between state and code.
func computeAttributeDiffs(stateNode, codeNode parser.Node) []parser.AttributeDiff {
	var diffs []parser.AttributeDiff

	for key, codeVal := range codeNode.Data.Attributes {
		// Skip internal metadata keys
		if strings.HasPrefix(key, "_") {
			continue
		}

		stateVal, exists := stateNode.Data.Attributes[key]
		if !exists {
			diffs = append(diffs, parser.AttributeDiff{
				Field:      key,
				StateValue: "<missing>",
				CodeValue:  codeVal,
			})
			continue
		}

		if !attributesEqual(codeVal, stateVal) {
			diffs = append(diffs, parser.AttributeDiff{
				Field:      key,
				StateValue: stateVal,
				CodeValue:  codeVal,
			})
		}
	}

	return diffs
}

func compareDeps(stateTargets, codeTargets []string) bool {
	sort.Strings(stateTargets)
	sort.Strings(codeTargets)
	return slicesEqual(stateTargets, codeTargets)
}

func getAttrString(attrs map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := attrs[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func getNameTag(attrs map[string]interface{}) string {
	if tags, ok := attrs["tags"].(map[string]interface{}); ok {
		if name, ok := tags["Name"].(string); ok {
			return name
		}
	}
	if tagsAll, ok := attrs["tags_all"].(map[string]interface{}); ok {
		if name, ok := tagsAll["Name"].(string); ok {
			return name
		}
	}
	return ""
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func attributesEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if reflect.DeepEqual(a, b) {
		return true
	}

	// Structural comparison via JSON normalization
	ja, errA := json.Marshal(a)
	jb, errB := json.Marshal(b)
	if errA == nil && errB == nil && string(ja) == string(jb) {
		return true
	}

	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
