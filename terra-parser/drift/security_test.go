package drift

import (
	"os"
	"path/filepath"
	"testing"

	"terra-parser/parser"
)

func TestSecurityScanVulnerableInfrastructure(t *testing.T) {
	// Create temporary directory with deliberately vulnerable Terraform resources
	tmpDir, err := os.MkdirTemp("", "terra_sec_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	vulnerableTF := `
# Deliberately vulnerable S3 bucket (no encryption, no access block)
resource "aws_s3_bucket" "insecure_data" {
  bucket = "insecure-customer-data-bucket"
}

# Deliberately vulnerable RDS instance (storage_encrypted = false)
resource "aws_db_instance" "unencrypted_rds" {
  allocated_storage = 20
  engine            = "postgres"
  instance_class    = "db.t3.micro"
  storage_encrypted = false
}

# Standard VPC (missing flow logs)
resource "aws_vpc" "app_vpc" {
  cidr_block = "10.0.0.0/16"
}
`
	mainTfPath := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(mainTfPath, []byte(vulnerableTF), 0644); err != nil {
		t.Fatalf("failed to write main.tf: %v", err)
	}

	// 1. Run Security Scan
	issues, err := RunSecurityScan(tmpDir)
	if err != nil {
		t.Fatalf("RunSecurityScan returned unexpected error: %v", err)
	}

	if len(issues) == 0 {
		t.Fatalf("expected security issues to be detected, got 0")
	}

	// 2. Verify S3 bucket issues
	s3Issues, hasS3 := issues["aws_s3_bucket.insecure_data"]
	if !hasS3 || len(s3Issues) == 0 {
		t.Errorf("expected security issues for aws_s3_bucket.insecure_data, got none. Full issues: %v", issues)
	} else {
		if s3Issues[0].Severity != "HIGH" && s3Issues[0].Severity != "MEDIUM" && s3Issues[0].Severity != "LOW" && s3Issues[0].Severity != "CRITICAL" {
			t.Errorf("invalid severity on S3 issue: %s", s3Issues[0].Severity)
		}
		if s3Issues[0].RuleID == "" {
			t.Errorf("empty RuleID on S3 issue")
		}
	}

	// 3. Verify RDS unencrypted issues
	rdsIssues, hasRDS := issues["aws_db_instance.unencrypted_rds"]
	if !hasRDS || len(rdsIssues) == 0 {
		t.Errorf("expected security issues for aws_db_instance.unencrypted_rds, got none")
	}

	// 4. Verify AttachSecurityIssues integrates into GraphResponse
	graph, err := parser.ParseHCLDirectory(tmpDir)
	if err != nil {
		t.Fatalf("failed to parse HCL directory: %v", err)
	}

	AttachSecurityIssues(graph, issues)

	var s3Node *parser.Node
	for _, n := range graph.Nodes {
		if n.ID == "aws_s3_bucket.insecure_data" {
			s3Node = &n
			break
		}
	}

	if s3Node == nil {
		t.Fatalf("aws_s3_bucket.insecure_data node not found in graph")
	}

	if len(s3Node.Data.SecurityIssues) == 0 {
		t.Errorf("expected SecurityIssues on s3Node to be populated, got 0")
	}
}
