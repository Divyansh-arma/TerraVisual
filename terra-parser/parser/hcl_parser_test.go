package parser

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseHCLDirectory(t *testing.T) {
	mockTFDir := filepath.Join("..", "testdata", "mock_tf")
	mockStateFile := filepath.Join("..", "testdata", "mock_tfstate.json")

	hclGraph, err := ParseHCLDirectory(mockTFDir)
	if err != nil {
		t.Fatalf("ParseHCLDirectory failed: %v", err)
	}

	stateGraph, err := ParseTFStateFile(mockStateFile)
	if err != nil {
		t.Fatalf("ParseTFStateFile failed: %v", err)
	}

	// 1. Verify Node count and structure
	if len(hclGraph.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(hclGraph.Nodes))
	}

	if len(hclGraph.Edges) != 3 {
		t.Fatalf("expected 3 edges, got %d", len(hclGraph.Edges))
	}

	// 2. Assert Node properties and graph contract compliance
	for i, hclNode := range hclGraph.Nodes {
		stateNode := stateGraph.Nodes[i]
		if hclNode.ID != stateNode.ID {
			t.Errorf("node %d: expected ID '%s', got '%s'", i, stateNode.ID, hclNode.ID)
		}
		if hclNode.Type != stateNode.Type {
			t.Errorf("node %s: expected type '%s', got '%s'", hclNode.ID, stateNode.Type, hclNode.Type)
		}
		if hclNode.Data.Label != stateNode.Data.Label {
			t.Errorf("node %s: expected label '%s', got '%s'", hclNode.ID, stateNode.Data.Label, hclNode.Data.Label)
		}
		if hclNode.Data.Provider != stateNode.Data.Provider {
			t.Errorf("node %s: expected provider '%s', got '%s'", hclNode.ID, stateNode.Data.Provider, hclNode.Data.Provider)
		}
		if hclNode.Data.ResourceType != stateNode.Data.ResourceType {
			t.Errorf("node %s: expected resourceType '%s', got '%s'", hclNode.ID, stateNode.Data.ResourceType, hclNode.Data.ResourceType)
		}
		if hclNode.Data.Module != stateNode.Data.Module {
			t.Errorf("node %s: expected module '%s', got '%s'", hclNode.ID, stateNode.Data.Module, hclNode.Data.Module)
		}
		if hclNode.Data.IsDataSource != stateNode.Data.IsDataSource {
			t.Errorf("node %s: expected isDataSource %v, got %v", hclNode.ID, stateNode.Data.IsDataSource, hclNode.Data.IsDataSource)
		}
		if hclNode.Data.DriftStatus != stateNode.Data.DriftStatus {
			t.Errorf("node %s: expected driftStatus '%s', got '%s'", hclNode.ID, stateNode.Data.DriftStatus, hclNode.Data.DriftStatus)
		}
		if hclNode.ParentID != stateNode.ParentID {
			t.Errorf("node %s: expected ParentID '%s', got '%s'", hclNode.ID, stateNode.ParentID, hclNode.ParentID)
		}
	}

	// 3. Verify that VPC parent appears first in nodes list
	if hclGraph.Nodes[0].ID != "aws_vpc.main" {
		t.Errorf("expected first node to be parent 'aws_vpc.main', got '%s'", hclGraph.Nodes[0].ID)
	}

	// 4. Verify subnet parentID is aws_vpc.main
	var foundSubnet bool
	for _, n := range hclGraph.Nodes {
		if n.ID == "aws_subnet.public" {
			foundSubnet = true
			if n.ParentID != "aws_vpc.main" {
				t.Errorf("expected aws_subnet.public ParentID to be 'aws_vpc.main', got '%s'", n.ParentID)
			}
		}
	}
	if !foundSubnet {
		t.Errorf("aws_subnet.public not found in hclGraph")
	}

	// 3. Assert Edges match state graph edges
	if !reflect.DeepEqual(hclGraph.Edges, stateGraph.Edges) {
		hclEdgesJSON, _ := json.MarshalIndent(hclGraph.Edges, "", "  ")
		stateEdgesJSON, _ := json.MarshalIndent(stateGraph.Edges, "", "  ")
		t.Errorf("HCL edges do not match state edges.\nHCL:\n%s\nState:\n%s", hclEdgesJSON, stateEdgesJSON)
	}
}

func TestInferProviderFromResourceType(t *testing.T) {
	tests := []struct {
		resType  string
		expected string
	}{
		{"aws_vpc", "aws"},
		{"google_compute_instance", "google"},
		{"azurerm_resource_group", "azurerm"},
		{"random_string", "random"},
		{"kubernetes_namespace", "kubernetes"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		got := inferProviderFromResourceType(tt.resType)
		if got != tt.expected {
			t.Errorf("inferProviderFromResourceType(%q) = %q, expected %q", tt.resType, got, tt.expected)
		}
	}
}
