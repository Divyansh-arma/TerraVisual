package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTFStateMockFile(t *testing.T) {
	// Locate mock tfstate file
	mockPath := filepath.Join("..", "testdata", "mock_tfstate.json")
	file, err := os.Open(mockPath)
	if err != nil {
		t.Fatalf("failed to open mock tfstate file: %v", err)
	}
	defer file.Close()

	graph, err := ParseTFState(file)
	if err != nil {
		t.Fatalf("ParseTFState failed: %v", err)
	}

	// 1. Verify Nodes
	if len(graph.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(graph.Nodes))
	}

	expectedNodes := map[string]struct {
		Label        string
		Provider     string
		ResourceType string
		Module       string
		IsDataSource bool
		DriftStatus  string
	}{
		"aws_vpc.main": {
			Label:        "main",
			Provider:     "aws",
			ResourceType: "aws_vpc",
			Module:       "root",
			IsDataSource: false,
			DriftStatus:  "unknown",
		},
		"aws_subnet.public": {
			Label:        "public",
			Provider:     "aws",
			ResourceType: "aws_subnet",
			Module:       "root",
			IsDataSource: false,
			DriftStatus:  "unknown",
		},
		"aws_instance.web": {
			Label:        "web",
			Provider:     "aws",
			ResourceType: "aws_instance",
			Module:       "root",
			IsDataSource: false,
			DriftStatus:  "unknown",
		},
	}

	for _, node := range graph.Nodes {
		expected, ok := expectedNodes[node.ID]
		if !ok {
			t.Errorf("unexpected node ID: %s", node.ID)
			continue
		}

		if node.Type != "infrastructureNode" {
			t.Errorf("node %s: expected type 'infrastructureNode', got '%s'", node.ID, node.Type)
		}

		if node.Position.X != 0 || node.Position.Y != 0 {
			t.Errorf("node %s: expected position (0, 0), got (%f, %f)", node.ID, node.Position.X, node.Position.Y)
		}

		if node.Data.Label != expected.Label {
			t.Errorf("node %s: expected label '%s', got '%s'", node.ID, expected.Label, node.Data.Label)
		}

		if node.Data.Provider != expected.Provider {
			t.Errorf("node %s: expected provider '%s', got '%s'", node.ID, expected.Provider, node.Data.Provider)
		}

		if node.Data.ResourceType != expected.ResourceType {
			t.Errorf("node %s: expected resourceType '%s', got '%s'", node.ID, expected.ResourceType, node.Data.ResourceType)
		}

		if node.Data.Module != expected.Module {
			t.Errorf("node %s: expected module '%s', got '%s'", node.ID, expected.Module, node.Data.Module)
		}

		if node.Data.IsDataSource != expected.IsDataSource {
			t.Errorf("node %s: expected isDataSource %v, got %v", node.ID, expected.IsDataSource, node.Data.IsDataSource)
		}

		if node.Data.DriftStatus != expected.DriftStatus {
			t.Errorf("node %s: expected driftStatus '%s', got '%s'", node.ID, expected.DriftStatus, node.Data.DriftStatus)
		}
		if node.ID == "aws_subnet.public" && node.ParentID != "aws_vpc.main" {
			t.Errorf("expected aws_subnet.public ParentID to be 'aws_vpc.main', got '%s'", node.ParentID)
		}
	}

	// Verify that parent node appears first in the list
	if graph.Nodes[0].ID != "aws_vpc.main" {
		t.Errorf("expected first node to be parent 'aws_vpc.main', got '%s'", graph.Nodes[0].ID)
	}

	// 2. Verify Edges
	expectedEdges := map[string]struct {
		Source string
		Target string
	}{
		"e-aws_subnet.public-aws_vpc.main": {
			Source: "aws_subnet.public",
			Target: "aws_vpc.main",
		},
		"e-aws_instance.web-aws_subnet.public": {
			Source: "aws_instance.web",
			Target: "aws_subnet.public",
		},
		"e-aws_instance.web-aws_vpc.main": {
			Source: "aws_instance.web",
			Target: "aws_vpc.main",
		},
	}

	if len(graph.Edges) != len(expectedEdges) {
		t.Fatalf("expected %d edges, got %d", len(expectedEdges), len(graph.Edges))
	}

	for _, edge := range graph.Edges {
		expected, ok := expectedEdges[edge.ID]
		if !ok {
			t.Errorf("unexpected edge ID: %s", edge.ID)
			continue
		}

		if edge.Source != expected.Source {
			t.Errorf("edge %s: expected source '%s', got '%s'", edge.ID, expected.Source, edge.Source)
		}

		if edge.Target != expected.Target {
			t.Errorf("edge %s: expected target '%s', got '%s'", edge.ID, expected.Target, edge.Target)
		}

		if edge.Type != "smoothstep" {
			t.Errorf("edge %s: expected type 'smoothstep', got '%s'", edge.ID, edge.Type)
		}

		if !edge.Animated {
			t.Errorf("edge %s: expected animated true, got false", edge.ID)
		}
	}

	// 3. Verify JSON serialization matches exact contract schema
	jsonBytes, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal graph to JSON: %v", err)
	}

	var unmarshaledMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &unmarshaledMap); err != nil {
		t.Fatalf("failed to unmarshal generated JSON: %v", err)
	}

	if _, ok := unmarshaledMap["nodes"]; !ok {
		t.Errorf("missing 'nodes' top-level key in marshaled JSON")
	}

	if _, ok := unmarshaledMap["edges"]; !ok {
		t.Errorf("missing 'edges' top-level key in marshaled JSON")
	}
}

func TestParseTFStateComplex(t *testing.T) {
	complexJSON := `{
  "version": 4,
  "terraform_version": "1.7.0",
  "resources": [
    {
      "module": "module.database",
      "mode": "managed",
      "type": "aws_db_instance",
      "name": "primary",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"].us_west_2",
      "instances": [
        {
          "dependencies": [
            "module.database.aws_db_subnet_group.default[0]",
            "aws_security_group.db"
          ]
        }
      ]
    },
    {
      "mode": "data",
      "type": "aws_ami",
      "name": "ubuntu",
      "provider": "provider.aws",
      "instances": [{}]
    }
  ]
}`

	graph, err := ParseTFState(strings.NewReader(complexJSON))
	if err != nil {
		t.Fatalf("ParseTFState failed: %v", err)
	}

	if len(graph.Nodes) != 3 {
		t.Fatalf("expected 3 nodes (including module container), got %d", len(graph.Nodes))
	}

	var dataNode, moduleChildNode, moduleContainerNode Node
	for _, n := range graph.Nodes {
		if n.ID == "aws_ami.ubuntu" {
			dataNode = n
		} else if n.ID == "module.database.aws_db_instance.primary" {
			moduleChildNode = n
		} else if n.ID == "module.database" {
			moduleContainerNode = n
		}
	}

	if !dataNode.Data.IsDataSource {
		t.Errorf("expected aws_ami.ubuntu to have isDataSource=true")
	}

	if moduleContainerNode.Data.ResourceType != "module" {
		t.Errorf("expected module.database to have ResourceType=module")
	}

	if moduleChildNode.ParentID != "module.database" {
		t.Errorf("expected module child to have ParentID=module.database, got %s", moduleChildNode.ParentID)
	}

	if moduleChildNode.Data.Module != "module.database" {
		t.Errorf("expected module to be 'module.database', got '%s'", moduleChildNode.Data.Module)
	}
	if moduleChildNode.Data.Provider != "aws" {
		t.Errorf("expected provider to be 'aws', got '%s'", moduleChildNode.Data.Provider)
	}

	// Verify Edges from indexed dependencies
	expectedEdgeIDs := map[string]bool{
		"e-module.database.aws_db_instance.primary-module.database.aws_db_subnet_group.default": true,
		"e-module.database.aws_db_instance.primary-aws_security_group.db":                        true,
	}

	for _, edge := range graph.Edges {
		if !expectedEdgeIDs[edge.ID] {
			t.Errorf("unexpected edge: %s", edge.ID)
		}
	}
}

func TestExtractProviderName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`provider["registry.terraform.io/hashicorp/aws"]`, "aws"},
		{`provider["registry.terraform.io/hashicorp/google"]`, "google"},
		{`provider["registry.terraform.io/hashicorp/azurerm"]`, "azurerm"},
		{`provider["aws"]`, "aws"},
		{`provider.aws`, "aws"},
		{`provider.aws.west`, "aws"},
		{`aws`, "aws"},
		{``, "unknown"},
	}

	for _, tt := range tests {
		result := ExtractProviderName(tt.input)
		if result != tt.expected {
			t.Errorf("ExtractProviderName(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}
