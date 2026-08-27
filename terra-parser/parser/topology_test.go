package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAWSTopologyHierarchy(t *testing.T) {
	tmpDir := t.TempDir()

	hclContent := `
resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "public_1a" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.0.1.0/24"
  availability_zone       = "us-west-2a"
  map_public_ip_on_launch = true
  tags = {
    Name = "prod-public-1a"
  }
}

resource "aws_subnet" "private_1b" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.2.0/24"
  availability_zone = "us-west-2b"
  tags = {
    Name = "prod-private-1b"
  }
}

resource "aws_instance" "web" {
  ami           = "ami-123456"
  instance_type = "t3.micro"
  subnet_id     = aws_subnet.public_1a.id
}

resource "aws_db_instance" "db" {
  allocated_storage = 20
  engine            = "postgres"
  subnet_id         = aws_subnet.private_1b.id
}

resource "aws_internet_gateway" "gw" {
  vpc_id = aws_vpc.main.id
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

	// 1. Verify Synthetic AZ Nodes
	az1a, hasAZ1a := nodeMap["az-us-west-2a"]
	if !hasAZ1a {
		t.Errorf("Expected synthetic AZ node 'az-us-west-2a' to exist, got nodes: %+v", graph.Nodes)
	} else {
		if az1a.Data.ResourceType != "aws_availability_zone" {
			t.Errorf("Expected AZ resourceType to be 'aws_availability_zone', got %s", az1a.Data.ResourceType)
		}
		if az1a.ParentID != "aws_vpc.main" {
			t.Errorf("Expected AZ ParentID to be 'aws_vpc.main', got %s", az1a.ParentID)
		}
		if !az1a.Data.IsContainer {
			t.Errorf("Expected AZ to have isContainer=true")
		}
	}

	az1b, hasAZ1b := nodeMap["az-us-west-2b"]
	if !hasAZ1b {
		t.Errorf("Expected synthetic AZ node 'az-us-west-2b' to exist")
	} else {
		if az1b.ParentID != "aws_vpc.main" {
			t.Errorf("Expected AZ 1b ParentID to be 'aws_vpc.main', got %s", az1b.ParentID)
		}
	}

	// 2. Verify Subnets attached to synthetic AZs
	subnet1a, hasSubnet1a := nodeMap["aws_subnet.public_1a"]
	if !hasSubnet1a {
		t.Errorf("Expected aws_subnet.public_1a to exist")
	} else {
		if subnet1a.ParentID != "az-us-west-2a" {
			t.Errorf("Expected subnet1a ParentID to be 'az-us-west-2a', got %s", subnet1a.ParentID)
		}
		if !subnet1a.Data.IsContainer {
			t.Errorf("Expected subnet1a with EC2 instance to have isContainer=true")
		}
		if subnet1a.Data.Attributes["subnet_type"] != "public" {
			t.Errorf("Expected subnet1a to be marked as public subnet, got %v", subnet1a.Data.Attributes["subnet_type"])
		}
	}

	subnet1b, hasSubnet1b := nodeMap["aws_subnet.private_1b"]
	if !hasSubnet1b {
		t.Errorf("Expected aws_subnet.private_1b to exist")
	} else {
		if subnet1b.ParentID != "az-us-west-2b" {
			t.Errorf("Expected subnet1b ParentID to be 'az-us-west-2b', got %s", subnet1b.ParentID)
		}
		if !subnet1b.Data.IsContainer {
			t.Errorf("Expected subnet1b with DB instance to have isContainer=true")
		}
		if subnet1b.Data.Attributes["subnet_type"] != "private" {
			t.Errorf("Expected subnet1b to be marked as private subnet, got %v", subnet1b.Data.Attributes["subnet_type"])
		}
	}

	// 3. Verify Compute & Database attached to Subnets
	ec2, hasEC2 := nodeMap["aws_instance.web"]
	if !hasEC2 {
		t.Errorf("Expected aws_instance.web to exist")
	} else {
		if ec2.ParentID != "aws_subnet.public_1a" {
			t.Errorf("Expected aws_instance.web ParentID to be 'aws_subnet.public_1a', got %s", ec2.ParentID)
		}
	}

	rds, hasRDS := nodeMap["aws_db_instance.db"]
	if !hasRDS {
		t.Errorf("Expected aws_db_instance.db to exist")
	} else {
		if rds.ParentID != "aws_subnet.private_1b" {
			t.Errorf("Expected aws_db_instance.db ParentID to be 'aws_subnet.private_1b', got %s", rds.ParentID)
		}
	}

	// 4. Verify Internet Gateway attached directly to VPC
	igw, hasIGW := nodeMap["aws_internet_gateway.gw"]
	if !hasIGW {
		t.Errorf("Expected aws_internet_gateway.gw to exist")
	} else {
		if igw.ParentID != "aws_vpc.main" {
			t.Errorf("Expected aws_internet_gateway.gw ParentID to be 'aws_vpc.main', got %s", igw.ParentID)
		}
	}

	// 5. Verify Topological Sort (Parents before children across multi-level hierarchy)
	indexMap := make(map[string]int)
	for i, n := range graph.Nodes {
		indexMap[n.ID] = i
	}

	// VPC < AZ < Subnet < Resource
	vpcIdx := indexMap["aws_vpc.main"]
	azIdx := indexMap["az-us-west-2a"]
	subIdx := indexMap["aws_subnet.public_1a"]
	ec2Idx := indexMap["aws_instance.web"]

	if !(vpcIdx < azIdx && azIdx < subIdx && subIdx < ec2Idx) {
		t.Errorf("Topological hierarchy order violated: VPC(idx %d) < AZ(idx %d) < Subnet(idx %d) < EC2(idx %d)", vpcIdx, azIdx, subIdx, ec2Idx)
	}
}
