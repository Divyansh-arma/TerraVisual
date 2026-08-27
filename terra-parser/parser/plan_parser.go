package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// TFPlan represents the top-level Terraform Plan JSON output (terraform show -json).
type TFPlan struct {
	FormatVersion    string             `json:"format_version"`
	TerraformVersion string             `json:"terraform_version"`
	ResourceChanges  []TFResourceChange `json:"resource_changes"`
	Configuration    *TFConfiguration   `json:"configuration,omitempty"`
}

// TFResourceChange represents an individual resource modification in the plan.
type TFResourceChange struct {
	Address        string      `json:"address"`
	ModuleAddress  string      `json:"module_address,omitempty"`
	Mode           string      `json:"mode"`
	Type           string      `json:"type"`
	Name           string      `json:"name"`
	Index          interface{} `json:"index,omitempty"`
	ProviderName   string      `json:"provider_name"`
	Change         TFChange    `json:"change"`
	ActionReason   string      `json:"action_reason,omitempty"`
	PreviousAddress string     `json:"previous_address,omitempty"`
}

// TFChange contains the before/after state and action list for a resource.
type TFChange struct {
	Actions      []string               `json:"actions"`
	Before       map[string]interface{} `json:"before,omitempty"`
	After        map[string]interface{} `json:"after,omitempty"`
	AfterUnknown map[string]interface{} `json:"after_unknown,omitempty"`
}

// TFConfiguration represents the static configuration tree embedded in the plan.
type TFConfiguration struct {
	RootModule *TFConfigModule `json:"root_module,omitempty"`
}

// TFConfigModule contains resources and child module calls in the configuration.
type TFConfigModule struct {
	Resources   []TFConfigResource            `json:"resources,omitempty"`
	ModuleCalls map[string]TFConfigModuleCall `json:"module_calls,omitempty"`
}

// TFConfigResource describes references and dependencies declared in configuration.
type TFConfigResource struct {
	Address     string                  `json:"address"`
	Type        string                  `json:"type"`
	Name        string                  `json:"name"`
	Expressions map[string]TFConfigExpr `json:"expressions,omitempty"`
	DependsOn   []string                `json:"depends_on,omitempty"`
}

// TFConfigExpr captures attribute reference targets.
type TFConfigExpr struct {
	References []string `json:"references,omitempty"`
}

// TFConfigModuleCall captures child module expressions.
type TFConfigModuleCall struct {
	Source      string                  `json:"source"`
	Expressions map[string]TFConfigExpr `json:"expressions,omitempty"`
	Module      *TFConfigModule         `json:"module,omitempty"`
}

// ParseTFPlan reads a Terraform Plan JSON stream and builds a visual GraphResponse.
func ParseTFPlan(reader io.Reader) (*GraphResponse, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read terraform plan json: %w", err)
	}

	var plan TFPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse terraform plan json: %w", err)
	}

	nodeMap := make(map[string]Node)
	edgeMap := make(map[string]Edge)

	// 1. Build Nodes from resource_changes
	for _, rc := range plan.ResourceChanges {
		// Ignore data sources from primary topology unless desired
		isDataSource := rc.Mode == "data"
		if isDataSource {
			continue
		}

		nodeID := rc.Address
		provider := ExtractProviderName(rc.ProviderName)
		if provider == "unknown" {
			provider = inferProviderFromResourceType(rc.Type)
		}

		driftStatus := mapPlanActionsToStatus(rc.Change.Actions)

		// Merge attributes: prioritize after, fallback to before
		attributes := make(map[string]interface{})
		if rc.Change.Before != nil {
			for k, v := range rc.Change.Before {
				attributes[k] = v
			}
		}
		if rc.Change.After != nil {
			for k, v := range rc.Change.After {
				attributes[k] = v
			}
		}

		// Compute readable label
		label := rc.Name
		if tags, ok := attributes["tags"].(map[string]interface{}); ok {
			if nameTag, ok := tags["Name"].(string); ok && nameTag != "" {
				label = nameTag
			}
		} else if tagsAll, ok := attributes["tags_all"].(map[string]interface{}); ok {
			if nameTag, ok := tagsAll["Name"].(string); ok && nameTag != "" {
				label = nameTag
			}
		}
		if label == rc.Name && rc.Type != "" {
			label = fmt.Sprintf("%s.%s", rc.Type, rc.Name)
		}

		node := Node{
			ID:   nodeID,
			Type: "infrastructureNode",
			Position: Position{
				X: 0,
				Y: 0,
			},
			Data: NodeData{
				Label:        label,
				Provider:     provider,
				ResourceType: rc.Type,
				Module:       rc.ModuleAddress,
				IsDataSource: isDataSource,
				IsContainer:  rc.Type == "aws_vpc" || rc.Type == "aws_availability_zone" || rc.Type == "azurerm_virtual_network",
				DriftStatus:  driftStatus,
				Attributes:   attributes,
			},
			ParentID: "",
		}

		nodeMap[nodeID] = node
	}

	// 2. Build Dependency Edges from Configuration
	if plan.Configuration != nil && plan.Configuration.RootModule != nil {
		collectConfigEdges(plan.Configuration.RootModule, nodeMap, edgeMap)
	}

	// 3. Fallback attribute-based edges (vpc_id, subnet_id, etc.)
	for srcID, srcNode := range nodeMap {
		attrs := srcNode.Data.Attributes
		if attrs == nil {
			continue
		}

		// vpc_id reference
		if vpcVal, ok := attrs["vpc_id"].(string); ok && vpcVal != "" {
			if targetID := findMatchingNodeID(vpcVal, nodeMap); targetID != "" && targetID != srcID {
				edgeID := fmt.Sprintf("e-%s-%s", targetID, srcID)
				edgeMap[edgeID] = Edge{
					ID:       edgeID,
					Source:   targetID,
					Target:   srcID,
					Type:     "smoothstep",
					Animated: true,
				}
			}
		}

		// subnet_id reference
		if subVal, ok := attrs["subnet_id"].(string); ok && subVal != "" {
			if targetID := findMatchingNodeID(subVal, nodeMap); targetID != "" && targetID != srcID {
				edgeID := fmt.Sprintf("e-%s-%s", targetID, srcID)
				edgeMap[edgeID] = Edge{
					ID:       edgeID,
					Source:   targetID,
					Target:   srcID,
					Type:     "smoothstep",
					Animated: true,
				}
			}
		}
	}

	// 4. Apply AWS Architecture Hierarchy Topology (VPC -> AZ Columns -> Subnets -> Compute)
	nodeMap, edgeMap = ApplyAWSTopology(nodeMap, edgeMap)

	nodes := make([]Node, 0, len(nodeMap))
	for _, n := range nodeMap {
		nodes = append(nodes, n)
	}

	edges := make([]Edge, 0, len(edgeMap))
	for _, e := range edgeMap {
		edges = append(edges, e)
	}

	SortNodesParentsFirst(nodes)
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].ID < edges[j].ID
	})

	return &GraphResponse{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// ParseTFPlanFile reads and parses a Terraform Plan JSON from the given file path.
func ParseTFPlanFile(filePath string) (*GraphResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plan file %s: %w", filePath, err)
	}
	defer file.Close()

	return ParseTFPlan(file)
}

func mapPlanActionsToStatus(actions []string) string {
	if len(actions) == 0 {
		return "IN_SYNC"
	}

	hasCreate := false
	hasDelete := false
	hasUpdate := false
	hasNoOp := false

	for _, a := range actions {
		switch a {
		case "create":
			hasCreate = true
		case "delete":
			hasDelete = true
		case "update":
			hasUpdate = true
		case "no-op", "read":
			hasNoOp = true
		}
	}

	if hasCreate && hasDelete {
		return "MODIFIED" // Replace
	}
	if hasCreate {
		return "CREATE"
	}
	if hasDelete {
		return "DESTROY"
	}
	if hasUpdate {
		return "MODIFIED"
	}
	if hasNoOp {
		return "IN_SYNC"
	}

	return "IN_SYNC"
}

func collectConfigEdges(mod *TFConfigModule, nodeMap map[string]Node, edgeMap map[string]Edge) {
	if mod == nil {
		return
	}

	for _, res := range mod.Resources {
		targetID := res.Address
		if _, exists := nodeMap[targetID]; !exists {
			continue
		}

		// Check explicit depends_on
		for _, dep := range res.DependsOn {
			if srcID := findMatchingNodeID(dep, nodeMap); srcID != "" && srcID != targetID {
				edgeID := fmt.Sprintf("e-%s-%s", srcID, targetID)
				edgeMap[edgeID] = Edge{
					ID:       edgeID,
					Source:   srcID,
					Target:   targetID,
					Type:     "smoothstep",
					Animated: true,
				}
			}
		}

		// Check attribute expressions references
		for _, expr := range res.Expressions {
			for _, ref := range expr.References {
				if srcID := findMatchingNodeID(ref, nodeMap); srcID != "" && srcID != targetID {
					edgeID := fmt.Sprintf("e-%s-%s", srcID, targetID)
					edgeMap[edgeID] = Edge{
						ID:       edgeID,
						Source:   srcID,
						Target:   targetID,
						Type:     "smoothstep",
						Animated: true,
					}
				}
			}
		}
	}

	// Recurse into child modules
	for _, call := range mod.ModuleCalls {
		if call.Module != nil {
			collectConfigEdges(call.Module, nodeMap, edgeMap)
		}
	}
}

func findMatchingNodeID(ref string, nodeMap map[string]Node) string {
	if _, exists := nodeMap[ref]; exists {
		return ref
	}

	// Strip attribute suffixes e.g. aws_vpc.main.id -> aws_vpc.main
	parts := strings.Split(ref, ".")
	if len(parts) >= 2 {
		candidate := parts[0] + "." + parts[1]
		if _, exists := nodeMap[candidate]; exists {
			return candidate
		}
	}

	for id := range nodeMap {
		if strings.HasPrefix(ref, id) || strings.HasPrefix(id, ref) {
			return id
		}
	}

	return ""
}
