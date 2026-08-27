package drift

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"terra-parser/parser"
)

// CheckovOutput represents the JSON structure produced by checkov -o json.
type CheckovOutput struct {
	Results struct {
		FailedChecks []struct {
			CheckID      string `json:"check_id"`
			CheckName    string `json:"check_name"`
			Resource     string `json:"resource"`
			Severity     string `json:"severity"`
			Guideline    string `json:"guideline"`
			ShortDesc    string `json:"description"`
		} `json:"failed_checks"`
	} `json:"results"`
}

// TrivyOutput represents the JSON structure produced by trivy config -f json.
type TrivyOutput struct {
	Results []struct {
		Target            string `json:"Target"`
		Misconfigurations []struct {
			ID            string `json:"ID"`
			Title         string `json:"Title"`
			Description   string `json:"Description"`
			Message       string `json:"Message"`
			Severity      string `json:"Severity"`
			Status        string `json:"Status"`
			CauseMetadata struct {
				Resource string `json:"Resource"`
			} `json:"CauseMetadata"`
		} `json:"Misconfigurations"`
	} `json:"Results"`
}

// RunSecurityScan executes Checkov or Trivy (or falls back to built-in rules) to scan codePath.
// Completely resilient: will never panic or block parsing if external tools fail.
func RunSecurityScan(codePath string) (map[string][]parser.SecurityIssue, error) {
	issuesMap := make(map[string][]parser.SecurityIssue)

	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "[SecurityScan] Recovered from unexpected panic: %v\n", r)
		}
	}()

	if codePath == "" {
		return issuesMap, nil
	}

	// 1. Try Checkov if available
	if checkovPath, err := exec.LookPath("checkov"); err == nil && checkovPath != "" {
		cmd := exec.Command(checkovPath, "-d", codePath, "-o", "json", "--compact", "--quiet", "--framework", "terraform")
		output, cmdErr := cmd.Output()
		if cmdErr == nil && len(output) > 0 {
			var checkovRes CheckovOutput
			if err := json.Unmarshal(output, &checkovRes); err == nil && len(checkovRes.Results.FailedChecks) > 0 {
				for _, fc := range checkovRes.Results.FailedChecks {
					resID := cleanResourceID(fc.Resource)
					sev := normalizeSeverity(fc.Severity)
					desc := fc.Guideline
					if desc == "" {
						desc = fc.ShortDesc
					}
					issuesMap[resID] = append(issuesMap[resID], parser.SecurityIssue{
						RuleID:      fc.CheckID,
						Severity:    sev,
						Title:       fc.CheckName,
						Description: desc,
					})
				}
				return issuesMap, nil
			}
		}
	}

	// 2. Try Trivy if available
	trivyPath := findTrivyBinary()
	if trivyPath != "" {
		cmd := exec.Command(trivyPath, "config", "-f", "json", codePath)
		output, cmdErr := cmd.Output()
		if cmdErr == nil && len(output) > 0 {
			var trivyRes TrivyOutput
			if err := json.Unmarshal(output, &trivyRes); err == nil {
				for _, r := range trivyRes.Results {
					for _, m := range r.Misconfigurations {
						if m.Status == "FAIL" && m.CauseMetadata.Resource != "" {
							resID := cleanResourceID(m.CauseMetadata.Resource)
							desc := m.Description
							if desc == "" {
								desc = m.Message
							}
							issuesMap[resID] = append(issuesMap[resID], parser.SecurityIssue{
								RuleID:      m.ID,
								Severity:    normalizeSeverity(m.Severity),
								Title:       m.Title,
								Description: desc,
							})
						}
					}
				}
				if len(issuesMap) > 0 {
					return issuesMap, nil
				}
			}
		}
	}

	// 3. Built-in Static Rules Engine (Fallback for air-gapped / test environments)
	builtinIssues := runBuiltinSecurityScan(codePath)
	for k, v := range builtinIssues {
		issuesMap[k] = append(issuesMap[k], v...)
	}

	return issuesMap, nil
}

// AttachSecurityIssues injects detected security issues into the matching nodes in a GraphResponse.
func AttachSecurityIssues(graph *parser.GraphResponse, issues map[string][]parser.SecurityIssue) {
	if graph == nil || len(issues) == 0 {
		return
	}

	for i := range graph.Nodes {
		nodeID := graph.Nodes[i].ID
		if nodeIssues, exists := issues[nodeID]; exists {
			graph.Nodes[i].Data.SecurityIssues = nodeIssues
		}
	}
}

func findTrivyBinary() string {
	candidates := []string{
		"trivy",
		"/opt/homebrew/bin/trivy",
		"/usr/local/bin/trivy",
		"/usr/bin/trivy",
	}
	for _, c := range candidates {
		if p, err := exec.LookPath(c); err == nil && p != "" {
			return p
		}
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func cleanResourceID(raw string) string {
	raw = strings.TrimSpace(raw)
	// Remove module prefix if in module (e.g. module.app.aws_s3_bucket.data -> aws_s3_bucket.data)
	if strings.Contains(raw, ".") {
		parts := strings.Split(raw, ".")
		if len(parts) >= 2 {
			return fmtResourceID(parts[len(parts)-2], parts[len(parts)-1])
		}
	}
	return raw
}

func fmtResourceID(resType, resName string) string {
	return strings.TrimSpace(resType) + "." + strings.TrimSpace(resName)
}

func normalizeSeverity(sev string) string {
	s := strings.ToUpper(strings.TrimSpace(sev))
	if s == "" {
		return "MEDIUM"
	}
	switch s {
	case "CRITICAL", "HIGH", "MEDIUM", "LOW":
		return s
	default:
		return "MEDIUM"
	}
}

// runBuiltinSecurityScan performs static checks on HCL files in codePath when external CLI is unavailable.
func runBuiltinSecurityScan(codePath string) map[string][]parser.SecurityIssue {
	issues := make(map[string][]parser.SecurityIssue)

	graph, err := parser.ParseHCLDirectory(codePath)
	if err != nil || graph == nil {
		return issues
	}

	for _, node := range graph.Nodes {
		resID := node.ID
		resType := node.Data.ResourceType
		attrs := node.Data.Attributes

		switch resType {
		case "aws_s3_bucket":
			if attrs == nil || (attrs["server_side_encryption_configuration"] == nil && attrs["bucket_prefix"] == nil) {
				issues[resID] = append(issues[resID], parser.SecurityIssue{
					RuleID:      "CKV_AWS_19",
					Severity:    "HIGH",
					Title:       "S3 Bucket has no server-side encryption configured",
					Description: "Ensure S3 bucket is encrypted with AWS KMS or AES-256 to protect sensitive data at rest.",
				})
			}
		case "aws_db_instance":
			if attrs == nil || attrs["storage_encrypted"] != true {
				issues[resID] = append(issues[resID], parser.SecurityIssue{
					RuleID:      "CKV_AWS_16",
					Severity:    "HIGH",
					Title:       "RDS DB Instance storage encryption is disabled",
					Description: "Ensure RDS database instances have storage encryption enabled.",
				})
			}
		case "aws_vpc":
			issues[resID] = append(issues[resID], parser.SecurityIssue{
				RuleID:      "AWS-0178",
				Severity:    "MEDIUM",
				Title:       "VPC Flow Logs are not enabled",
				Description: "VPC Flow Logs provide network traffic visibility for threat detection and compliance.",
			})
		case "google_sql_database_instance":
			if attrs == nil || attrs["require_ssl"] != true {
				issues[resID] = append(issues[resID], parser.SecurityIssue{
					RuleID:      "CKV_GCP_14",
					Severity:    "HIGH",
					Title:       "Cloud SQL instance allows non-SSL connections",
					Description: "Ensure Cloud SQL database instances enforce SSL/TLS encryption in transit.",
				})
			}
		case "azurerm_storage_account":
			if attrs == nil || attrs["enable_https_traffic_only"] == false {
				issues[resID] = append(issues[resID], parser.SecurityIssue{
					RuleID:      "CKV_AZURE_3",
					Severity:    "HIGH",
					Title:       "Azure Storage Account allows unencrypted HTTP traffic",
					Description: "Ensure Storage Accounts require secure transfer (HTTPS) for all API requests.",
				})
			}
		}
	}

	return issues
}
