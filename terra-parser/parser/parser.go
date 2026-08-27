package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

var (
	providerBracketRegex = regexp.MustCompile(`\["([^"]+)"\]`)
	arrayIndexRegex      = regexp.MustCompile(`\[[^\]]+\]`)
)

// ParseTFStateFile reads a terraform.tfstate file from the filesystem.
func ParseTFStateFile(filePath string) (*GraphResponse, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open state file: %w", err)
	}
	defer f.Close()

	return ParseTFState(f)
}

// ParseTFState parses a Terraform state JSON stream into GraphResponse.
func ParseTFState(r io.Reader) (*GraphResponse, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read state data: %w", err)
	}

	var state TFState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state json: %w", err)
	}

	nodeMap := make(map[string]Node)
	edgeMap := make(map[string]Edge)

	// Build VPC ID lookup table so child resources can resolve their parent VPC container
	vpcLookup := make(map[string]string)
	for _, res := range state.Resources {
		if res.Type == "aws_vpc" || res.Type == "azurerm_virtual_network" || res.Type == "google_compute_network" {
			moduleName := res.Module
			if moduleName == "" {
				moduleName = "root"
			}
			var nodeID string
			if moduleName == "root" {
				nodeID = fmt.Sprintf("%s.%s", res.Type, res.Name)
			} else {
				nodeID = fmt.Sprintf("%s.%s.%s", moduleName, res.Type, res.Name)
			}
			vpcLookup[nodeID] = nodeID
			for _, inst := range res.Instances {
				if idVal, ok := inst.Attributes["id"].(string); ok && idVal != "" {
					vpcLookup[idVal] = nodeID
				}
			}
		}
	}

	// First pass: Discover and unpack module containers from state
	for _, res := range state.Resources {
		if res.Module != "" && res.Module != "root" {
			modID := res.Module
			if _, exists := nodeMap[modID]; !exists {
				nodeMap[modID] = Node{
					ID:   modID,
					Type: "infrastructureNode",
					Position: Position{
						X: 0,
						Y: 0,
					},
					Data: NodeData{
						Label:        modID,
						Provider:     "terraform",
						ResourceType: "module",
						Module:       "root",
						IsDataSource: false,
						IsContainer:  true,
						DriftStatus:  "unknown",
						Attributes:   make(map[string]interface{}),
					},
				}
			}
		}
	}

	for _, res := range state.Resources {
		// Only process managed resources and data sources
		isDataSource := res.Mode == "data"

		moduleName := res.Module
		if moduleName == "" {
			moduleName = "root"
		}

		var nodeID string
		if moduleName == "root" {
			nodeID = fmt.Sprintf("%s.%s", res.Type, res.Name)
		} else {
			nodeID = fmt.Sprintf("%s.%s.%s", moduleName, res.Type, res.Name)
		}

		provider := ExtractProviderName(res.Provider)

		var attributes map[string]interface{}
		if len(res.Instances) > 0 && res.Instances[0].Attributes != nil {
			attributes = res.Instances[0].Attributes
		}

		var parentID string
		if attributes != nil {
			if vpcIDVal, ok := attributes["vpc_id"].(string); ok && vpcIDVal != "" {
				if mapped, exists := vpcLookup[vpcIDVal]; exists {
					parentID = mapped
				} else if strings.HasPrefix(vpcIDVal, "aws_vpc.") {
					parts := strings.Split(vpcIDVal, ".")
					if len(parts) >= 2 {
						parentID = fmt.Sprintf("%s.%s", parts[0], parts[1])
					}
				}
			}
		}

		// If resource has no VPC parent but is inside a module, attach to the module container
		if parentID == "" && moduleName != "root" {
			parentID = moduleName
		}

		node := Node{
			ID:   nodeID,
			Type: "infrastructureNode",
			Position: Position{
				X: 0,
				Y: 0,
			},
			Data: NodeData{
				Label:        res.Name,
				Provider:     provider,
				ResourceType: res.Type,
				Module:       moduleName,
				IsDataSource: isDataSource,
				DriftStatus:  "unknown",
				Attributes:   attributes,
			},
			ParentID: parentID,
		}

		nodeMap[nodeID] = node

		// Collect dependencies from resource and instance levels
		deps := make([]string, 0)
		deps = append(deps, res.Dependencies...)
		for _, inst := range res.Instances {
			deps = append(deps, inst.Dependencies...)
		}

		for _, rawDep := range deps {
			cleanDep := cleanDependency(rawDep)
			if cleanDep == "" || cleanDep == nodeID {
				continue
			}

			edgeID := fmt.Sprintf("e-%s-%s", nodeID, cleanDep)
			edgeMap[edgeID] = Edge{
				ID:       edgeID,
				Source:   nodeID,
				Target:   cleanDep,
				Type:     "smoothstep",
				Animated: true,
			}
		}
	}

	// Apply AWS Architecture Hierarchy Topology
	nodeMap, edgeMap = ApplyAWSTopology(nodeMap, edgeMap)

	nodes := make([]Node, 0, len(nodeMap))
	for _, node := range nodeMap {
		nodes = append(nodes, node)
	}

	edges := make([]Edge, 0, len(edgeMap))
	for _, edge := range edgeMap {
		edges = append(edges, edge)
	}

	// Sort nodes topologically with parents before children
	SortNodesParentsFirst(nodes)

	sort.Slice(edges, func(i, j int) bool {
		return edges[i].ID < edges[j].ID
	})

	return &GraphResponse{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// SortNodesParentsFirst ensures parent nodes always appear before their children in the slice,
// which is required by React Flow for compound subgraphs of arbitrary nesting depth.
func SortNodesParentsFirst(nodes []Node) {
	nodeMap := make(map[string]Node, len(nodes))
	for _, n := range nodes {
		nodeMap[n.ID] = n
	}

	// Calculate ancestor depth for every node
	getDepth := func(id string) int {
		depth := 0
		curr := id
		visited := make(map[string]bool)
		for {
			if visited[curr] {
				break
			}
			visited[curr] = true
			parentID := nodeMap[curr].ParentID
			if parentID == "" {
				break
			}
			depth++
			curr = parentID
		}
		return depth
	}

	nodeDepths := make(map[string]int, len(nodes))
	for _, n := range nodes {
		nodeDepths[n.ID] = getDepth(n.ID)
	}

	sort.Slice(nodes, func(i, j int) bool {
		a, b := nodes[i], nodes[j]
		depthA := nodeDepths[a.ID]
		depthB := nodeDepths[b.ID]

		// Ancestors with smaller depth come first
		if depthA != depthB {
			return depthA < depthB
		}

		return a.ID < b.ID
	})
}

// ExtractProviderName extracts the short provider name (e.g., "aws", "google", "azure")
// from various Terraform provider string formats.
func ExtractProviderName(providerConfig string) string {
	if providerConfig == "" {
		return "unknown"
	}

	// Case 1: provider["registry.terraform.io/hashicorp/aws"] or provider["aws"]
	if matches := providerBracketRegex.FindStringSubmatch(providerConfig); len(matches) > 1 {
		p := matches[1]
		parts := strings.Split(p, "/")
		return parts[len(parts)-1]
	}

	// Case 2: provider.aws or provider.aws.alias
	trimmed := strings.TrimPrefix(providerConfig, "provider.")
	parts := strings.Split(trimmed, ".")
	if len(parts) > 0 && parts[0] != "" {
		slashParts := strings.Split(parts[0], "/")
		return slashParts[len(slashParts)-1]
	}

	return strings.ToLower(providerConfig)
}

// cleanDependency normalizes a dependency reference string.
// Example: "aws_vpc.main[0]" -> "aws_vpc.main"
func cleanDependency(dep string) string {
	dep = strings.TrimSpace(dep)
	if dep == "" {
		return ""
	}

	// Remove index like [0] or ["key"]
	dep = arrayIndexRegex.ReplaceAllString(dep, "")

	return dep
}
