package drift

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"terra-parser/parser"
)

// DriftStatus constants
const (
	StatusInSync         = "IN_SYNC"
	StatusModified       = "MODIFIED"
	StatusMissingInState = "MISSING_IN_STATE"
	StatusMissingInCode  = "MISSING_IN_CODE"
)

// DetectDrift compares the stateGraph against the codeGraph and returns a unified GraphResponse
// with driftStatus assigned to every resource node and combined dependency edges.
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

	allIDs := make(map[string]bool)
	for id := range stateNodeMap {
		allIDs[id] = true
	}
	for id := range codeNodeMap {
		allIDs[id] = true
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

	mergedNodes := make([]parser.Node, 0, len(allIDs))

	for id := range allIDs {
		stateNode, inState := stateNodeMap[id]
		codeNode, inCode := codeNodeMap[id]

		if inCode && !inState {
			// Resource defined in code but missing from state
			node := codeNode
			node.Data.DriftStatus = StatusMissingInState
			mergedNodes = append(mergedNodes, node)
		} else if inState && !inCode {
			// Resource present in state but deleted from code
			node := stateNode
			node.Data.DriftStatus = StatusMissingInCode
			mergedNodes = append(mergedNodes, node)
		} else {
			// Resource exists in both state and code -> check for modifications
			isModified := hasDrift(stateNode, codeNode, stateDeps[id], codeDeps[id])

			driftStatus := StatusInSync
			if isModified {
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

			mergedNode := stateNode
			mergedNode.ParentID = parentID
			mergedNode.Data.Attributes = mergedAttrs
			mergedNode.Data.DriftStatus = driftStatus
			mergedNodes = append(mergedNodes, mergedNode)
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

// hasDrift checks if a resource's attributes or dependencies differ between state and code.
func hasDrift(stateNode, codeNode parser.Node, stateTargets, codeTargets []string) bool {
	// 1. Compare dependencies
	sort.Strings(stateTargets)
	sort.Strings(codeTargets)
	if !slicesEqual(stateTargets, codeTargets) {
		return true
	}

	// 2. Compare declared code attributes against state attributes
	for key, codeVal := range codeNode.Data.Attributes {
		stateVal, exists := stateNode.Data.Attributes[key]
		if !exists {
			return true
		}
		if !attributesEqual(codeVal, stateVal) {
			return true
		}
	}

	return false
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
