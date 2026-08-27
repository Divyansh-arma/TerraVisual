package hclsync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"terra-parser/parser"
)

func TestSyncGraphToCode(t *testing.T) {
	// Create temporary directory for isolated test
	tmpDir, err := os.MkdirTemp("", "terra_sync_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	initialTF := `
# Main VPC configuration
resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

# Public subnet to be deleted
resource "aws_subnet" "public" {
  vpc_id     = aws_vpc.main.id
  cidr_block = "10.0.1.0/24"
}
`
	mainTfPath := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(mainTfPath, []byte(initialTF), 0644); err != nil {
		t.Fatalf("failed to write initial main.tf: %v", err)
	}

	// Modified graph:
	// - aws_vpc.main is preserved
	// - aws_subnet.public is removed (missing from graph)
	// - aws_s3_bucket.new_bucket is added
	modifiedGraphJSON := `{
  "nodes": [
    {
      "id": "aws_vpc.main",
      "type": "infrastructureNode",
      "position": { "x": 0, "y": 0 },
      "data": {
        "label": "main",
        "provider": "aws",
        "resourceType": "aws_vpc",
        "module": "root",
        "isDataSource": false,
        "driftStatus": "IN_SYNC",
        "attributes": {
          "cidr_block": "10.0.0.0/16"
        }
      }
    },
    {
      "id": "aws_s3_bucket.new_bucket",
      "type": "infrastructureNode",
      "position": { "x": 100, "y": 100 },
      "data": {
        "label": "new_bucket",
        "provider": "aws",
        "resourceType": "aws_s3_bucket",
        "module": "root",
        "isDataSource": false,
        "driftStatus": "MISSING_IN_STATE",
        "attributes": {
          "bucket": "my-test-app-bucket"
        }
      }
    }
  ],
  "edges": []
}`

	res, err := SyncGraphToCode(strings.NewReader(modifiedGraphJSON), tmpDir)
	if err != nil {
		t.Fatalf("SyncGraphToCode failed: %v", err)
	}

	if res.Status != "success" {
		t.Errorf("expected status 'success', got '%s'", res.Status)
	}

	if len(res.AddedNodes) != 1 || res.AddedNodes[0] != "aws_s3_bucket.new_bucket" {
		t.Errorf("unexpected AddedNodes: %v", res.AddedNodes)
	}

	if len(res.RemovedNodes) != 1 || res.RemovedNodes[0] != "aws_subnet.public" {
		t.Errorf("unexpected RemovedNodes: %v", res.RemovedNodes)
	}

	// Verify rewritten file contents
	updatedBytes, err := os.ReadFile(mainTfPath)
	if err != nil {
		t.Fatalf("failed to read updated main.tf: %v", err)
	}
	content := string(updatedBytes)

	if strings.Contains(content, "aws_subnet") {
		t.Errorf("expected aws_subnet to be removed from main.tf, but found:\n%s", content)
	}

	if !strings.Contains(content, "aws_vpc") {
		t.Errorf("expected aws_vpc to be preserved in main.tf, but not found:\n%s", content)
	}

	if !strings.Contains(content, "aws_s3_bucket") {
		t.Errorf("expected aws_s3_bucket to be added to main.tf, but not found:\n%s", content)
	}

	// Verify that the updated code parses cleanly back into a GraphResponse
	graph, err := parser.ParseHCLDirectory(tmpDir)
	if err != nil {
		t.Fatalf("ParseHCLDirectory failed on updated code: %v", err)
	}

	if len(graph.Nodes) != 2 {
		t.Fatalf("expected 2 nodes in parsed graph, got %d", len(graph.Nodes))
	}

	nodeIDs := map[string]bool{
		graph.Nodes[0].ID: true,
		graph.Nodes[1].ID: true,
	}

	if !nodeIDs["aws_vpc.main"] || !nodeIDs["aws_s3_bucket.new_bucket"] {
		t.Errorf("unexpected nodes in re-parsed graph: %v", nodeIDs)
	}
}

func TestSyncMultiCloudResources(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "terra_multicloud_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	graphJSON := `{
  "nodes": [
    {
      "id": "azurerm_virtual_network.primary_vnet",
      "type": "infrastructureNode",
      "data": {
        "label": "primary_vnet",
        "provider": "azure",
        "resourceType": "azurerm_virtual_network",
        "attributes": {
          "address_space": ["10.0.0.0/16"],
          "location": "eastus"
        }
      }
    },
    {
      "id": "google_sql_database_instance.pg_db",
      "type": "infrastructureNode",
      "data": {
        "label": "pg_db",
        "provider": "gcp",
        "resourceType": "google_sql_database_instance",
        "attributes": {
          "database_version": "POSTGRES_15",
          "tier": "db-f1-micro"
        }
      }
    },
    {
      "id": "aws_db_instance.app_db",
      "type": "infrastructureNode",
      "data": {
        "label": "app_db",
        "provider": "aws",
        "resourceType": "aws_db_instance",
        "attributes": {
          "engine": "postgres",
          "instance_class": "db.t3.micro",
          "allocated_storage": 20
        }
      }
    }
  ],
  "edges": []
}`

	res, err := SyncGraphToCode(strings.NewReader(graphJSON), tmpDir)
	if err != nil {
		t.Fatalf("SyncGraphToCode failed: %v", err)
	}

	if len(res.AddedNodes) != 3 {
		t.Errorf("expected 3 added nodes, got %d", len(res.AddedNodes))
	}

	mainTfPath := filepath.Join(tmpDir, "main.tf")
	contentBytes, err := os.ReadFile(mainTfPath)
	if err != nil {
		t.Fatalf("failed to read main.tf: %v", err)
	}
	content := string(contentBytes)

	if !strings.Contains(content, "azurerm_virtual_network") {
		t.Errorf("expected azurerm_virtual_network in main.tf, got:\n%s", content)
	}
	if !strings.Contains(content, "google_sql_database_instance") {
		t.Errorf("expected google_sql_database_instance in main.tf, got:\n%s", content)
	}
	if !strings.Contains(content, "aws_db_instance") {
		t.Errorf("expected aws_db_instance in main.tf, got:\n%s", content)
	}

	// Verify HCL directory parses back correctly
	parsedGraph, err := parser.ParseHCLDirectory(tmpDir)
	if err != nil {
		t.Fatalf("failed to parse generated multi-cloud HCL: %v", err)
	}

	if len(parsedGraph.Nodes) != 3 {
		t.Fatalf("expected 3 parsed nodes, got %d", len(parsedGraph.Nodes))
	}
}

