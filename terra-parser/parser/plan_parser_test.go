package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTFPlanJSON(t *testing.T) {
	tmpDir := t.TempDir()

	planJSON := `{
  "format_version": "1.2",
  "terraform_version": "1.7.5",
  "resource_changes": [
    {
      "address": "aws_vpc.production",
      "mode": "managed",
      "type": "aws_vpc",
      "name": "production",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["create"],
        "before": null,
        "after": {
          "cidr_block": "10.0.0.0/16",
          "tags": { "Name": "production-vpc" }
        }
      }
    },
    {
      "address": "aws_subnet.public_1",
      "mode": "managed",
      "type": "aws_subnet",
      "name": "public_1",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["create"],
        "before": null,
        "after": {
          "availability_zone": "us-east-1a",
          "cidr_block": "10.0.1.0/24",
          "vpc_id": "aws_vpc.production",
          "tags": { "Name": "Public Subnet 1" }
        }
      }
    },
    {
      "address": "aws_subnet.private_1",
      "mode": "managed",
      "type": "aws_subnet",
      "name": "private_1",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["update"],
        "before": {
          "cidr_block": "10.0.2.0/24"
        },
        "after": {
          "availability_zone": "us-east-1a",
          "cidr_block": "10.0.2.0/24",
          "vpc_id": "aws_vpc.production",
          "tags": { "Name": "Private Subnet 1" }
        }
      }
    },
    {
      "address": "aws_s3_bucket.legacy_logs",
      "mode": "managed",
      "type": "aws_s3_bucket",
      "name": "legacy_logs",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["delete"],
        "before": {
          "bucket": "company-legacy-logs"
        },
        "after": null
      }
    },
    {
      "address": "aws_dynamodb_table.app_state",
      "mode": "managed",
      "type": "aws_dynamodb_table",
      "name": "app_state",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["no-op"],
        "before": {
          "name": "app-state-table"
        },
        "after": {
          "name": "app-state-table"
        }
      }
    }
  ],
  "configuration": {
    "root_module": {
      "resources": [
        {
          "address": "aws_subnet.public_1",
          "type": "aws_subnet",
          "name": "public_1",
          "expressions": {
            "vpc_id": {
              "references": ["aws_vpc.production.id"]
            }
          }
        }
      ]
    }
  }
}`

	planFilePath := filepath.Join(tmpDir, "tfplan.json")
	if err := os.WriteFile(planFilePath, []byte(planJSON), 0644); err != nil {
		t.Fatalf("Failed to write plan file: %v", err)
	}

	graph, err := ParseTFPlanFile(planFilePath)
	if err != nil {
		t.Fatalf("ParseTFPlanFile failed: %v", err)
	}

	nodeMap := make(map[string]Node)
	for _, n := range graph.Nodes {
		nodeMap[n.ID] = n
	}

	// 1. Verify VPC node and CREATE status
	vpcNode, ok := nodeMap["aws_vpc.production"]
	if !ok {
		t.Fatalf("Expected aws_vpc.production to exist")
	}
	if vpcNode.Data.DriftStatus != "CREATE" {
		t.Errorf("Expected VPC drift status CREATE, got %s", vpcNode.Data.DriftStatus)
	}

	// 2. Verify Update Subnet (MODIFIED status)
	privSubnet, ok := nodeMap["aws_subnet.private_1"]
	if !ok {
		t.Fatalf("Expected aws_subnet.private_1 to exist")
	}
	if privSubnet.Data.DriftStatus != "MODIFIED" {
		t.Errorf("Expected private subnet drift status MODIFIED, got %s", privSubnet.Data.DriftStatus)
	}

	// 3. Verify Delete S3 Bucket (DESTROY status)
	s3Node, ok := nodeMap["aws_s3_bucket.legacy_logs"]
	if !ok {
		t.Fatalf("Expected aws_s3_bucket.legacy_logs to exist")
	}
	if s3Node.Data.DriftStatus != "DESTROY" {
		t.Errorf("Expected S3 bucket drift status DESTROY, got %s", s3Node.Data.DriftStatus)
	}

	// 4. Verify No-op DynamoDB (IN_SYNC status)
	dbNode, ok := nodeMap["aws_dynamodb_table.app_state"]
	if !ok {
		t.Fatalf("Expected aws_dynamodb_table.app_state to exist")
	}
	if dbNode.Data.DriftStatus != "IN_SYNC" {
		t.Errorf("Expected DynamoDB drift status IN_SYNC, got %s", dbNode.Data.DriftStatus)
	}

	// 5. Verify dependency edge from VPC to Subnet
	var hasEdge bool
	for _, e := range graph.Edges {
		if strings.Contains(e.Source, "aws_vpc.production") && strings.Contains(e.Target, "aws_subnet.public_1") {
			hasEdge = true
			break
		}
	}
	if !hasEdge {
		t.Logf("Edges present: %+v", graph.Edges)
	}
}
