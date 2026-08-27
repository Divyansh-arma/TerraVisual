package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"terra-parser/drift"
	"terra-parser/hclsync"
	"terra-parser/parser"
)

func main() {
	var statePath string
	var codePath string
	var syncPath string
	var noSecurity bool

	flag.StringVar(&statePath, "state", "", "Path to terraform.tfstate file")
	flag.StringVar(&codePath, "code", "", "Path to Terraform HCL directory (.tf files)")
	flag.StringVar(&syncPath, "sync", "", "Path to Terraform HCL directory to sync incoming Graph JSON to")
	flag.BoolVar(&noSecurity, "no-security", false, "Disable automated local security scanning")
	flag.Parse()

	args := flag.Args()

	// Mode 1: Bi-directional Graph to Code Sync
	if syncPath != "" {
		result, err := hclsync.SyncGraphToCode(os.Stdin, syncPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error syncing graph to code in %s: %v\n", syncPath, err)
			os.Exit(1)
		}
		output, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(output))
		return
	}

	// Positional arguments drift mode support: terra-parser <state> <code> or terra-parser <code> <state>
	if statePath == "" && codePath == "" && len(args) >= 2 {
		arg1, arg2 := args[0], args[1]
		info1, err1 := os.Stat(arg1)
		info2, err2 := os.Stat(arg2)
		if err1 == nil && err2 == nil {
			if info1.IsDir() && !info2.IsDir() {
				codePath = arg1
				statePath = arg2
			} else {
				statePath = arg1
				codePath = arg2
			}
		} else {
			statePath = arg1
			codePath = arg2
		}
	}

	var graph *parser.GraphResponse
	var err error
	var scanTargetDir string

	if statePath != "" && codePath != "" {
		// Mode 2: Drift detection mode
		stateGraph, sErr := parser.ParseTFStateFile(statePath)
		if sErr != nil {
			fmt.Fprintf(os.Stderr, "Error parsing state file %s: %v\n", statePath, sErr)
			os.Exit(1)
		}

		codeGraph, cErr := parser.ParseHCLDirectory(codePath)
		if cErr != nil {
			fmt.Fprintf(os.Stderr, "Error parsing HCL directory %s: %v\n", codePath, cErr)
			os.Exit(1)
		}

		graph = drift.DetectDrift(stateGraph, codeGraph)
		scanTargetDir = codePath
	} else if len(args) == 1 && args[0] != "-" {
		inputPath := args[0]
		info, statErr := os.Stat(inputPath)
		if statErr != nil {
			fmt.Fprintf(os.Stderr, "Error accessing path %s: %v\n", inputPath, statErr)
			os.Exit(1)
		}

		if info.IsDir() {
			graph, err = parser.ParseHCLDirectory(inputPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing HCL directory %s: %v\n", inputPath, err)
				os.Exit(1)
			}
			scanTargetDir = inputPath
		} else {
			graph, err = parser.ParseTFStateFile(inputPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing tfstate file %s: %v\n", inputPath, err)
				os.Exit(1)
			}
		}
	} else {
		// Mode 3: Read tfstate from stdin
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 && len(args) == 0 && statePath == "" && codePath == "" && syncPath == "" {
			fmt.Fprintf(os.Stderr, "Usage:\n")
			fmt.Fprintf(os.Stderr, "  Single target: terra-parser <directory-or-tfstate-file>\n")
			fmt.Fprintf(os.Stderr, "  Drift mode:    terra-parser --state <tfstate-file> --code <hcl-dir>\n")
			fmt.Fprintf(os.Stderr, "                 terra-parser <tfstate-file> <hcl-dir>\n")
			fmt.Fprintf(os.Stderr, "  Sync mode:     terra-parser --sync <hcl-dir> (pipes graph JSON via stdin)\n")
			fmt.Fprintf(os.Stderr, "  Stdin pipe:    cat terraform.tfstate | terra-parser\n")
			os.Exit(1)
		}

		graph, err = parser.ParseTFState(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing tfstate from stdin: %v\n", err)
			os.Exit(1)
		}
	}

	// Run security scanning if code directory is present and not disabled
	if !noSecurity && scanTargetDir != "" && graph != nil {
		if issues, sErr := drift.RunSecurityScan(scanTargetDir); sErr == nil && len(issues) > 0 {
			drift.AttachSecurityIssues(graph, issues)
		}
	}

	output, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling graph output: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))
}
