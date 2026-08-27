package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnrollModularArchitectureWithTFVars(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Write terraform.tfvars
	tfvarsContent := `
vpc_cidr        = "10.0.0.0/16"
public_subnets  = ["10.0.1.0/24", "10.0.2.0/24"]
private_subnets = ["10.0.3.0/24", "10.0.4.0/24"]
eks_name        = "prod-eks-cluster"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "terraform.tfvars"), []byte(tfvarsContent), 0644); err != nil {
		t.Fatalf("Failed to write terraform.tfvars: %v", err)
	}

	// 2. Write main.tf with variable references
	mainTFContent := `
variable "vpc_cidr" {
  type = string
}

variable "public_subnets" {
  type = list(string)
}

variable "private_subnets" {
  type = list(string)
}

variable "eks_name" {
  type = string
}

module "vpc" {
  source          = "terraform-aws-modules/vpc/aws"
  name            = "prod-vpc"
  cidr            = var.vpc_cidr
  public_subnets  = var.public_subnets
  private_subnets = var.private_subnets
}

module "eks" {
  source       = "terraform-aws-modules/eks/aws"
  cluster_name = var.eks_name
  vpc_id       = module.vpc.vpc_id
  subnet_ids   = module.vpc.private_subnets
}

module "s3_bucket" {
  source = "terraform-aws-modules/s3-bucket/aws"
  bucket = "prod-company-assets"
}

module "dynamodb_table" {
  source = "terraform-aws-modules/dynamodb-table/aws"
  name   = "prod-user-sessions"
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(mainTFContent), 0644); err != nil {
		t.Fatalf("Failed to write main.tf: %v", err)
	}

	// 3. Parse Directory
	graph, err := ParseHCLDirectory(tmpDir)
	if err != nil {
		t.Fatalf("ParseHCLDirectory failed: %v", err)
	}

	nodeMap := make(map[string]Node)
	for _, n := range graph.Nodes {
		nodeMap[n.ID] = n
	}

	// Assert VPC Parent Container
	vpcNode, hasVPC := nodeMap["vpc-main"]
	if !hasVPC {
		t.Fatalf("Expected vpc-main parent container to exist, got nodes: %+v", graph.Nodes)
	}
	if vpcNode.Data.ResourceType != "aws_vpc" {
		t.Errorf("Expected vpc-main ResourceType to be 'aws_vpc', got '%s'", vpcNode.Data.ResourceType)
	}
	if !vpcNode.Data.IsContainer {
		t.Errorf("Expected vpc-main to have isContainer=true")
	}
	if !strings.Contains(vpcNode.Data.Label, "10.0.0.0/16") {
		t.Errorf("Expected vpc-main label to contain resolved CIDR '10.0.0.0/16', got '%s'", vpcNode.Data.Label)
	}

	// Assert AZ Container Nodes
	az1, hasAZ1 := nodeMap["az-us-east-1a"]
	if !hasAZ1 {
		t.Fatalf("Expected az-us-east-1a container to exist")
	}
	if az1.ParentID != "vpc-main" {
		t.Errorf("Expected az-us-east-1a ParentID to be 'vpc-main', got '%s'", az1.ParentID)
	}

	az2, hasAZ2 := nodeMap["az-us-east-1b"]
	if !hasAZ2 {
		t.Fatalf("Expected az-us-east-1b container to exist")
	}
	if az2.ParentID != "vpc-main" {
		t.Errorf("Expected az-us-east-1b ParentID to be 'vpc-main', got '%s'", az2.ParentID)
	}

	// Assert Public and Private Subnet Child Nodes (nested inside respective AZs)
	pubSubnet1, hasPub1 := nodeMap["subnet-pub-1"]
	if !hasPub1 {
		t.Errorf("Expected subnet-pub-1 to exist")
	} else {
		if pubSubnet1.ParentID != "az-us-east-1a" {
			t.Errorf("Expected subnet-pub-1 ParentID to be 'az-us-east-1a', got '%s'", pubSubnet1.ParentID)
		}
		if !strings.Contains(pubSubnet1.Data.Label, "10.0.1.0/24") {
			t.Errorf("Expected subnet-pub-1 to contain '10.0.1.0/24', got '%s'", pubSubnet1.Data.Label)
		}
	}

	privSubnet1, hasPriv1 := nodeMap["subnet-priv-1"]
	if !hasPriv1 {
		t.Errorf("Expected subnet-priv-1 to exist")
	} else {
		if privSubnet1.ParentID != "az-us-east-1a" {
			t.Errorf("Expected subnet-priv-1 ParentID to be 'az-us-east-1a', got '%s'", privSubnet1.ParentID)
		}
		if !strings.Contains(privSubnet1.Data.Label, "10.0.3.0/24") {
			t.Errorf("Expected subnet-priv-1 to contain '10.0.3.0/24', got '%s'", privSubnet1.Data.Label)
		}
	}

	// Assert Internet Gateway
	igwNode, hasIGW := nodeMap["igw-main"]
	if !hasIGW {
		t.Errorf("Expected igw-main to exist")
	} else {
		if igwNode.ParentID != "vpc-main" {
			t.Errorf("Expected igw-main ParentID to be 'vpc-main', got '%s'", igwNode.ParentID)
		}
	}

	// Assert EKS Cluster & Node Group
	eksCluster, hasEKS := nodeMap["aws_eks_cluster"]
	if !hasEKS {
		t.Errorf("Expected aws_eks_cluster to exist")
	} else {
		if eksCluster.ParentID != "vpc-main" {
			t.Errorf("Expected aws_eks_cluster ParentID to be 'vpc-main', got '%s'", eksCluster.ParentID)
		}
		if !strings.Contains(eksCluster.Data.Label, "prod-eks-cluster") {
			t.Errorf("Expected EKS cluster to contain resolved name 'prod-eks-cluster', got '%s'", eksCluster.Data.Label)
		}
	}

	eksNG, hasNG := nodeMap["aws_eks_node_group"]
	if !hasNG {
		t.Errorf("Expected aws_eks_node_group to exist")
	} else {
		if eksNG.ParentID != "vpc-main" {
			t.Errorf("Expected aws_eks_node_group ParentID to be 'vpc-main', got '%s'", eksNG.ParentID)
		}
	}

	// Assert Standalone S3 & DynamoDB (outside VPC)
	s3Node, hasS3 := nodeMap["module.s3_bucket"]
	if !hasS3 {
		t.Errorf("Expected module.s3_bucket to exist")
	} else {
		if s3Node.Data.ResourceType != "aws_s3_bucket" {
			t.Errorf("Expected S3 ResourceType to be 'aws_s3_bucket', got '%s'", s3Node.Data.ResourceType)
		}
		if s3Node.ParentID != "" {
			t.Errorf("Expected s3-bucket ParentID to be '', got '%s'", s3Node.ParentID)
		}
		if s3Node.Data.Label != "prod-company-assets" {
			t.Errorf("Expected s3-bucket label to be 'prod-company-assets', got '%s'", s3Node.Data.Label)
		}
	}

	dbNode, hasDB := nodeMap["module.dynamodb_table"]
	if !hasDB {
		t.Errorf("Expected module.dynamodb_table to exist")
	} else {
		if dbNode.Data.ResourceType != "aws_dynamodb_table" {
			t.Errorf("Expected DynamoDB ResourceType to be 'aws_dynamodb_table', got '%s'", dbNode.Data.ResourceType)
		}
		if dbNode.ParentID != "" {
			t.Errorf("Expected dynamodb-table ParentID to be '', got '%s'", dbNode.ParentID)
		}
		if dbNode.Data.Label != "prod-user-sessions" {
			t.Errorf("Expected dynamodb-table label to be 'prod-user-sessions', got '%s'", dbNode.Data.Label)
		}
	}

	// Verify generic module wrapper cards were pruned
	if _, genericVPC := nodeMap["module.vpc"]; genericVPC {
		t.Errorf("Expected generic module.vpc card to be removed")
	}
	if _, genericEKS := nodeMap["module.eks"]; genericEKS {
		t.Errorf("Expected generic module.eks card to be removed")
	}

	// Verify Routing Edges:
	// 1. IGW -> Public Subnets
	// 2. Public Subnets -> EKS Cluster
	// 3. EKS Cluster -> Private Subnets (Node Groups)
	// 4. EKS Cluster -> Node Group
	var hasIGWToPub, hasPubToEKS, hasEKSToPriv, hasEKSToNG bool
	for _, e := range graph.Edges {
		if strings.Contains(e.Source, "igw-main") && strings.Contains(e.Target, "subnet-pub") {
			hasIGWToPub = true
		}
		if strings.Contains(e.Source, "subnet-pub") && strings.Contains(e.Target, "aws_eks_cluster") {
			hasPubToEKS = true
		}
		if strings.Contains(e.Source, "aws_eks_cluster") && strings.Contains(e.Target, "subnet-priv") {
			hasEKSToPriv = true
		}
		if strings.Contains(e.Source, "aws_eks_cluster") && strings.Contains(e.Target, "aws_eks_node_group") {
			hasEKSToNG = true
		}
	}

	if !hasIGWToPub {
		t.Errorf("Expected edge from IGW to public subnets")
	}
	if !hasPubToEKS {
		t.Errorf("Expected edge from public subnets to EKS cluster")
	}
	if !hasEKSToPriv {
		t.Errorf("Expected edge from EKS cluster to private subnets")
	}
	if !hasEKSToNG {
		t.Errorf("Expected edge from EKS cluster to EKS node group")
	}
}
