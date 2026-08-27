package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecomposeSemanticAWSModules(t *testing.T) {
	tmpDir := t.TempDir()

	hclContent := `
module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
  name   = "production-vpc"
  cidr   = "10.0.0.0/16"

  azs             = ["us-east-1a", "us-east-1b"]
  public_subnets  = ["10.0.1.0/24", "10.0.2.0/24"]
  private_subnets = ["10.0.10.0/24", "10.0.20.0/24"]

  enable_nat_gateway = true
}

module "eks" {
  source          = "terraform-aws-modules/eks/aws"
  cluster_name    = "production-eks"
  cluster_version = "1.29"

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets
}

module "s3_data_lake" {
  source = "terraform-aws-modules/s3-bucket/aws"
  bucket = "prod-data-lake-bucket"
}

module "app_state_table" {
  source = "terraform-aws-modules/dynamodb-table/aws"
  name   = "prod-app-state"
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(hclContent), 0644); err != nil {
		t.Fatalf("Failed to write main.tf: %v", err)
	}

	graph, err := ParseHCLDirectory(tmpDir)
	if err != nil {
		t.Fatalf("ParseHCLDirectory failed: %v", err)
	}

	nodeMap := make(map[string]Node)
	for _, n := range graph.Nodes {
		nodeMap[n.ID] = n
	}

	// 1. Verify VPC Container Node
	vpcNode, hasVPC := nodeMap["vpc-main"]
	if !hasVPC {
		t.Fatalf("Expected vpc-main to be created as VPC container node")
	}
	if vpcNode.Data.ResourceType != "aws_vpc" {
		t.Errorf("Expected vpc-main ResourceType to be 'aws_vpc', got '%s'", vpcNode.Data.ResourceType)
	}
	if !vpcNode.Data.IsContainer {
		t.Errorf("Expected VPC container node to have isContainer=true")
	}

	// 2. Verify Public & Private Subnets
	var publicSubnetCount, privateSubnetCount int
	for _, n := range graph.Nodes {
		if n.Data.ResourceType == "aws_subnet" {
			if n.Data.Attributes["subnet_type"] == "public" {
				publicSubnetCount++
			} else if n.Data.Attributes["subnet_type"] == "private" {
				privateSubnetCount++
			}
		}
	}

	if publicSubnetCount != 2 {
		t.Errorf("Expected 2 public subnets, got %d", publicSubnetCount)
	}
	if privateSubnetCount != 2 {
		t.Errorf("Expected 2 private subnets, got %d", privateSubnetCount)
	}

	// 3. Verify Internet Gateway
	hasIGW := false
	for _, n := range graph.Nodes {
		if n.Data.ResourceType == "aws_internet_gateway" {
			hasIGW = true
			break
		}
	}
	if !hasIGW {
		t.Errorf("Expected aws_internet_gateway to be synthesized")
	}

	// 4. Verify EKS Cluster & Node Group
	var hasEKSCluster, hasEKSNodeGroup bool
	for _, n := range graph.Nodes {
		if n.Data.ResourceType == "aws_eks_cluster" {
			hasEKSCluster = true
		}
		if n.Data.ResourceType == "aws_eks_node_group" {
			hasEKSNodeGroup = true
		}
	}

	if !hasEKSCluster {
		t.Errorf("Expected aws_eks_cluster to be synthesized")
	}
	if !hasEKSNodeGroup {
		t.Errorf("Expected aws_eks_node_group to be synthesized")
	}

	// 5. Verify Standalone S3 Bucket & DynamoDB Table
	s3Node, hasS3 := nodeMap["module.s3_data_lake"]
	if !hasS3 {
		t.Errorf("Expected module.s3_data_lake to exist")
	} else {
		if s3Node.Data.ResourceType != "aws_s3_bucket" {
			t.Errorf("Expected S3 ResourceType to be 'aws_s3_bucket', got '%s'", s3Node.Data.ResourceType)
		}
		if s3Node.ParentID != "" {
			t.Errorf("Expected standalone S3 bucket to have ParentID='', got '%s'", s3Node.ParentID)
		}
	}

	dbNode, hasDB := nodeMap["module.app_state_table"]
	if !hasDB {
		t.Errorf("Expected module.app_state_table to exist")
	} else {
		if dbNode.Data.ResourceType != "aws_dynamodb_table" {
			t.Errorf("Expected DynamoDB ResourceType to be 'aws_dynamodb_table', got '%s'", dbNode.Data.ResourceType)
		}
		if dbNode.ParentID != "" {
			t.Errorf("Expected standalone DynamoDB table to have ParentID='', got '%s'", dbNode.ParentID)
		}
	}

	// 6. Verify Edges: Public Subnet -> EKS -> Private Subnet & Node Group
	var hasClusterToNGEdge, hasPubToClusterEdge, hasClusterToPrivEdge bool
	for _, e := range graph.Edges {
		if strings.Contains(e.Source, "aws_eks_cluster") && strings.Contains(e.Target, "aws_eks_node_group") {
			hasClusterToNGEdge = true
		}
		if (strings.Contains(e.Source, "pub") || strings.Contains(e.Source, "public")) && strings.Contains(e.Target, "aws_eks_cluster") {
			hasPubToClusterEdge = true
		}
		if strings.Contains(e.Source, "aws_eks_cluster") && (strings.Contains(e.Target, "priv") || strings.Contains(e.Target, "private")) {
			hasClusterToPrivEdge = true
		}
	}

	if !hasClusterToNGEdge {
		t.Errorf("Expected edge from EKS cluster to EKS node group")
	}
	if !hasPubToClusterEdge {
		t.Errorf("Expected edge from public subnets to EKS cluster")
	}
	if !hasClusterToPrivEdge {
		t.Errorf("Expected edge from EKS cluster to private subnets")
	}
}
