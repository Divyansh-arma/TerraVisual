package drift

import (
	"path/filepath"
	"testing"

	"terra-parser/parser"
)

func TestDetectDrift(t *testing.T) {
	stateFilePath := filepath.Join("..", "testdata", "drift_test", "mock_state.json")
	codeDirPath := filepath.Join("..", "testdata", "drift_test", "code")

	stateGraph, err := parser.ParseTFStateFile(stateFilePath)
	if err != nil {
		t.Fatalf("ParseTFStateFile failed: %v", err)
	}

	codeGraph, err := parser.ParseHCLDirectory(codeDirPath)
	if err != nil {
		t.Fatalf("ParseHCLDirectory failed: %v", err)
	}

	mergedGraph := DetectDrift(stateGraph, codeGraph)

	if len(mergedGraph.Nodes) != 4 {
		t.Fatalf("expected 4 merged nodes, got %d", len(mergedGraph.Nodes))
	}

	expectedStatuses := map[string]string{
		"aws_vpc.main":        StatusModified,
		"aws_subnet.public":   StatusInSync,
		"aws_instance.legacy": StatusMissingInCode,
		"aws_s3_bucket.data":  StatusMissingInState,
	}

	for _, node := range mergedGraph.Nodes {
		expectedStatus, ok := expectedStatuses[node.ID]
		if !ok {
			t.Errorf("unexpected node ID in merged graph: %s", node.ID)
			continue
		}

		if node.Data.DriftStatus != expectedStatus {
			t.Errorf("node %s: expected driftStatus '%s', got '%s'", node.ID, expectedStatus, node.Data.DriftStatus)
		}
	}

	// Verify edges merged correctly (e.g. e-aws_subnet.public-aws_vpc.main exists)
	expectedEdgeID := "e-aws_subnet.public-aws_vpc.main"
	foundEdge := false
	for _, edge := range mergedGraph.Edges {
		if edge.ID == expectedEdgeID {
			foundEdge = true
			if edge.Source != "aws_subnet.public" || edge.Target != "aws_vpc.main" {
				t.Errorf("edge %s has incorrect source/target: (%s, %s)", edge.ID, edge.Source, edge.Target)
			}
		}
	}

	if !foundEdge {
		t.Errorf("expected edge %s in merged graph", expectedEdgeID)
	}
}

func TestDetectDriftDependencyMismatch(t *testing.T) {
	// Resource with same attributes but different dependencies -> MODIFIED
	stateGraph := &parser.GraphResponse{
		Nodes: []parser.Node{
			{
				ID:   "aws_subnet.test",
				Type: "infrastructureNode",
				Data: parser.NodeData{
					Label:        "test",
					ResourceType: "aws_subnet",
					Attributes:   map[string]interface{}{"cidr_block": "10.0.1.0/24"},
				},
			},
		},
		Edges: []parser.Edge{
			{
				ID:     "e-aws_subnet.test-aws_vpc.v1",
				Source: "aws_subnet.test",
				Target: "aws_vpc.v1",
			},
		},
	}

	codeGraph := &parser.GraphResponse{
		Nodes: []parser.Node{
			{
				ID:   "aws_subnet.test",
				Type: "infrastructureNode",
				Data: parser.NodeData{
					Label:        "test",
					ResourceType: "aws_subnet",
					Attributes:   map[string]interface{}{"cidr_block": "10.0.1.0/24"},
				},
			},
		},
		Edges: []parser.Edge{
			{
				ID:     "e-aws_subnet.test-aws_vpc.v2",
				Source: "aws_subnet.test",
				Target: "aws_vpc.v2",
			},
		},
	}

	merged := DetectDrift(stateGraph, codeGraph)
	if len(merged.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(merged.Nodes))
	}
	if merged.Nodes[0].Data.DriftStatus != StatusModified {
		t.Errorf("expected StatusModified for dependency mismatch, got %s", merged.Nodes[0].Data.DriftStatus)
	}
	if len(merged.Edges) != 2 {
		t.Errorf("expected 2 merged edges, got %d", len(merged.Edges))
	}
}

func TestFuzzyMatchDriftAndAttributeDiffs(t *testing.T) {
	// Subnet with different ID (module vs root) but identical CIDR should fuzzy match
	stateGraph := &parser.GraphResponse{
		Nodes: []parser.Node{
			{
				ID:   "module.vpc.aws_subnet.this[0]",
				Type: "infrastructureNode",
				Data: parser.NodeData{
					Label:        "public_subnet",
					ResourceType: "aws_subnet",
					Attributes: map[string]interface{}{
						"cidr_block":        "10.0.10.0/24",
						"availability_zone": "us-east-1a",
					},
				},
			},
		},
	}

	codeGraph := &parser.GraphResponse{
		Nodes: []parser.Node{
			{
				ID:   "aws_subnet.public_a",
				Type: "infrastructureNode",
				Data: parser.NodeData{
					Label:        "Public Subnet A",
					ResourceType: "aws_subnet",
					Attributes: map[string]interface{}{
						"cidr_block":        "10.0.10.0/24",
						"availability_zone": "us-east-1b", // changed from 1a to 1b
					},
				},
			},
		},
	}

	merged := DetectDrift(stateGraph, codeGraph)
	if len(merged.Nodes) != 1 {
		t.Fatalf("expected 1 fuzzy matched node, got %d", len(merged.Nodes))
	}

	node := merged.Nodes[0]
	if node.Data.DriftStatus != StatusModified {
		t.Errorf("expected StatusModified, got %s", node.Data.DriftStatus)
	}

	if len(node.Data.DriftDiffs) == 0 {
		t.Fatalf("expected DriftDiffs to contain attribute differences")
	}

	foundAZDiff := false
	for _, diff := range node.Data.DriftDiffs {
		if diff.Field == "availability_zone" {
			foundAZDiff = true
			if diff.StateValue != "us-east-1a" || diff.CodeValue != "us-east-1b" {
				t.Errorf("unexpected diff values: state=%v, code=%v", diff.StateValue, diff.CodeValue)
			}
		}
	}

	if !foundAZDiff {
		t.Errorf("expected availability_zone diff, got %+v", node.Data.DriftDiffs)
	}
}
