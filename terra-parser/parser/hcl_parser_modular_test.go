package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseHCLModularEnterprise(t *testing.T) {
	// Create a temporary modular Terraform workspace
	tmpDir := t.TempDir()

	// 1. Root main.tf
	rootTF := `
variable "env" {
  default = "production"
}

variable "subnets" {
  type    = list(string)
  default = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
}

module "vpc" {
  source      = "./modules/vpc"
  cidr_block  = "10.0.0.0/16"
  environment = var.env
}

module "eks" {
  source             = "./modules/eks"
  vpc_id             = module.vpc.vpc_id
  public_subnets     = slice(var.subnets, 0, length(var.subnets) > 2 ? 2 : 1)
  cluster_name       = "enterprise-cluster"
  enable_monitoring  = try(true, false)
  tags = {
    Project     = "Enterprise"
    Environment = lookup({ "prod" = "production" }, "prod", "default")
  }
}

resource "aws_route53_zone" "primary" {
  name = "example.com"
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(rootTF), 0644); err != nil {
		t.Fatalf("Failed to write root main.tf: %v", err)
	}

	// 2. ./modules/vpc/main.tf
	vpcDir := filepath.Join(tmpDir, "modules", "vpc")
	if err := os.MkdirAll(vpcDir, 0755); err != nil {
		t.Fatalf("Failed to create vpc module dir: %v", err)
	}
	vpcTF := `
variable "cidr_block" {}
variable "environment" {}

resource "aws_vpc" "this" {
  cidr_block           = var.cidr_block
  enable_dns_hostnames = true
}

resource "aws_subnet" "public" {
  vpc_id     = aws_vpc.this.id
  cidr_block = "10.0.1.0/24"
}
`
	if err := os.WriteFile(filepath.Join(vpcDir, "main.tf"), []byte(vpcTF), 0644); err != nil {
		t.Fatalf("Failed to write vpc main.tf: %v", err)
	}

	// 3. ./modules/eks/main.tf
	eksDir := filepath.Join(tmpDir, "modules", "eks")
	if err := os.MkdirAll(eksDir, 0755); err != nil {
		t.Fatalf("Failed to create eks module dir: %v", err)
	}
	eksTF := `
variable "vpc_id" {}
variable "public_subnets" {}
variable "cluster_name" {}
variable "enable_monitoring" {}
variable "tags" {}

resource "aws_eks_cluster" "main" {
  name     = var.cluster_name
  role_arn = "arn:aws:iam::123456789012:role/eks-role"

  vpc_config {
    subnet_ids = var.public_subnets
  }
}
`
	if err := os.WriteFile(filepath.Join(eksDir, "main.tf"), []byte(eksTF), 0644); err != nil {
		t.Fatalf("Failed to write eks main.tf: %v", err)
	}

	// 4. ./bootstrap/storage.tf
	bootstrapDir := filepath.Join(tmpDir, "bootstrap")
	if err := os.MkdirAll(bootstrapDir, 0755); err != nil {
		t.Fatalf("Failed to create bootstrap dir: %v", err)
	}
	bootstrapTF := `
resource "aws_s3_bucket" "tf_state" {
  bucket = "enterprise-tf-state"
}
`
	if err := os.WriteFile(filepath.Join(bootstrapDir, "storage.tf"), []byte(bootstrapTF), 0644); err != nil {
		t.Fatalf("Failed to write bootstrap storage.tf: %v", err)
	}

	// Run ParseHCLDirectory
	graph, err := ParseHCLDirectory(tmpDir)
	if err != nil {
		t.Fatalf("ParseHCLDirectory failed: %v", err)
	}

	if graph == nil {
		t.Fatal("Expected non-nil GraphResponse")
	}

	nodeMap := make(map[string]Node)
	for _, n := range graph.Nodes {
		nodeMap[n.ID] = n
	}

	// Verify Module Nodes
	vpcModNode, vpcModExists := nodeMap["module.vpc"]
	if !vpcModExists {
		t.Errorf("Expected module.vpc node to exist in graph, got nodes: %v", getKeys(nodeMap))
	} else {
		if vpcModNode.Data.ResourceType != "module" {
			t.Errorf("Expected module.vpc ResourceType to be 'module', got %s", vpcModNode.Data.ResourceType)
		}
		if vpcModNode.Data.Provider != "terraform" {
			t.Errorf("Expected module.vpc Provider to be 'terraform', got %s", vpcModNode.Data.Provider)
		}
	}

	eksModNode, eksModExists := nodeMap["module.eks"]
	if !eksModExists {
		t.Errorf("Expected module.eks node to exist in graph")
	} else {
		if eksModNode.Data.ResourceType != "module" {
			t.Errorf("Expected module.eks ResourceType to be 'module', got %s", eksModNode.Data.ResourceType)
		}
	}

	// Verify child resources attached to modules
	var foundVPCResource, foundSubnetResource, foundEKSResource bool
	for _, n := range graph.Nodes {
		if n.Data.ResourceType == "aws_vpc" && n.Data.Module == "module.vpc" {
			foundVPCResource = true
			if n.ParentID != "module.vpc" {
				t.Errorf("Expected aws_vpc inside module to have ParentID 'module.vpc', got '%s'", n.ParentID)
			}
		}
		if n.Data.ResourceType == "aws_subnet" && n.Data.Module == "module.vpc" {
			foundSubnetResource = true
		}
		if n.Data.ResourceType == "aws_eks_cluster" && n.Data.Module == "module.eks" {
			foundEKSResource = true
			if n.ParentID != "module.eks" {
				t.Errorf("Expected aws_eks_cluster inside module to have ParentID 'module.eks', got '%s'", n.ParentID)
			}
		}
	}

	if !foundVPCResource {
		t.Error("Expected aws_vpc in module.vpc to be parsed")
	}
	if !foundSubnetResource {
		t.Error("Expected aws_subnet in module.vpc to be parsed")
	}
	if !foundEKSResource {
		t.Error("Expected aws_eks_cluster in module.eks to be parsed")
	}

	// Verify Inter-Module edge (module.eks references module.vpc)
	var foundInterModuleEdge bool
	for _, edge := range graph.Edges {
		if (edge.Source == "module.eks" && edge.Target == "module.vpc") ||
			(edge.Source == "module.vpc" && edge.Target == "module.eks") {
			foundInterModuleEdge = true
			break
		}
	}

	if !foundInterModuleEdge {
		t.Errorf("Expected inter-module edge between module.eks and module.vpc, got edges: %+v", graph.Edges)
	}

	// Verify Parents First sorting order
	parentIndex := -1
	childIndex := -1
	for i, n := range graph.Nodes {
		if n.ID == "module.vpc" {
			parentIndex = i
		}
		if n.ParentID == "module.vpc" && childIndex == -1 {
			childIndex = i
		}
	}

	if parentIndex > childIndex && childIndex != -1 {
		t.Errorf("Expected parent node module.vpc (idx %d) to appear before child node (idx %d)", parentIndex, childIndex)
	}
}

func getKeys(m map[string]Node) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
