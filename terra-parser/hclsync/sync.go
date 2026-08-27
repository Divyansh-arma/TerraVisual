package hclsync

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"terra-parser/parser"
)

// SyncResult captures the summary of changes made to the HCL files.
type SyncResult struct {
	Status        string   `json:"status"`
	AddedNodes    []string `json:"addedNodes"`
	RemovedNodes  []string `json:"removedNodes"`
	ModifiedFiles []string `json:"modifiedFiles"`
}

// SyncGraphToCode synchronizes an incoming GraphResponse JSON to local HCL files in codeDir.
func SyncGraphToCode(r io.Reader, codeDir string) (*SyncResult, error) {
	var graph parser.GraphResponse
	if err := json.NewDecoder(r).Decode(&graph); err != nil {
		return nil, fmt.Errorf("failed to decode graph JSON: %w", err)
	}

	if err := os.MkdirAll(codeDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to access or create code directory %s: %w", codeDir, err)
	}

	graphResourceMap := make(map[string]parser.Node)
	for _, node := range graph.Nodes {
		graphResourceMap[node.ID] = node
	}

	entries, err := os.ReadDir(codeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read code directory %s: %w", codeDir, err)
	}

	var tfFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tf") && !strings.HasSuffix(entry.Name(), ".tf.json") {
			tfFiles = append(tfFiles, filepath.Join(codeDir, entry.Name()))
		}
	}
	sort.Strings(tfFiles)

	existingResources := make(map[string]bool)
	var removedNodes []string
	var modifiedFilesMap = make(map[string]bool)

	// Step 1: Deletion Logic - Inspect existing .tf files and remove blocks not present in graph
	for _, filePath := range tfFiles {
		src, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
		}

		file, diags := hclwrite.ParseConfig(src, filePath, hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			return nil, fmt.Errorf("failed to parse HCL file %s: %s", filePath, diags.Error())
		}

		body := file.Body()
		fileModified := false

		for _, block := range body.Blocks() {
			if block.Type() == "resource" {
				labels := block.Labels()
				if len(labels) == 2 {
					resID := fmt.Sprintf("%s.%s", labels[0], labels[1])
					if _, inGraph := graphResourceMap[resID]; !inGraph {
						// Resource was deleted in UI graph -> remove AST block
						body.RemoveBlock(block)
						removedNodes = append(removedNodes, resID)
						fileModified = true
					} else {
						existingResources[resID] = true
					}
				}
			}
		}

		if fileModified {
			if err := os.WriteFile(filePath, file.Bytes(), 0644); err != nil {
				return nil, fmt.Errorf("failed to write updated HCL file %s: %w", filePath, err)
			}
			modifiedFilesMap[filePath] = true
		}
	}

	// Step 2: Creation Logic - Identify new resources in graph not present in existing code
	var addedNodes []string
	mainTfPath := filepath.Join(codeDir, "main.tf")

	var mainFile *hclwrite.File
	if _, err := os.Stat(mainTfPath); err == nil {
		src, err := os.ReadFile(mainTfPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", mainTfPath, err)
		}
		mainFile, _ = hclwrite.ParseConfig(src, mainTfPath, hcl.Pos{Line: 1, Column: 1})
	}

	if mainFile == nil {
		mainFile = hclwrite.NewEmptyFile()
	}

	mainModified := false

	// Sort graph nodes for deterministic addition order
	sortedGraphNodes := make([]parser.Node, len(graph.Nodes))
	copy(sortedGraphNodes, graph.Nodes)
	sort.Slice(sortedGraphNodes, func(i, j int) bool {
		return sortedGraphNodes[i].ID < sortedGraphNodes[j].ID
	})

	for _, node := range sortedGraphNodes {
		if !existingResources[node.ID] {
			// Resource is newly added in graph -> create formatted HCL block
			resType := node.Data.ResourceType
			resName := node.Data.Label

			if resType == "" || resName == "" {
				parts := strings.Split(node.ID, ".")
				if len(parts) == 2 {
					resType = parts[0]
					resName = parts[1]
				} else {
					resType = "null_resource"
					resName = node.ID
				}
			}

			newBlock := mainFile.Body().AppendNewBlock("resource", []string{resType, resName})
			body := newBlock.Body()

			// Write attributes dynamically
			if len(node.Data.Attributes) > 0 {
				var attrKeys []string
				for k := range node.Data.Attributes {
					if k != "id" {
						attrKeys = append(attrKeys, k)
					}
				}
				sort.Strings(attrKeys)

				for _, k := range attrKeys {
					v := node.Data.Attributes[k]
					writeAttribute(body, k, v, graphResourceMap)
				}
			}

			addedNodes = append(addedNodes, node.ID)
			existingResources[node.ID] = true
			mainModified = true
		}
	}

	if mainModified {
		if err := os.WriteFile(mainTfPath, mainFile.Bytes(), 0644); err != nil {
			return nil, fmt.Errorf("failed to write %s: %w", mainTfPath, err)
		}
		modifiedFilesMap[mainTfPath] = true
	}

	var modifiedFiles []string
	for f := range modifiedFilesMap {
		modifiedFiles = append(modifiedFiles, f)
	}
	sort.Strings(modifiedFiles)

	return &SyncResult{
		Status:        "success",
		AddedNodes:    addedNodes,
		RemovedNodes:  removedNodes,
		ModifiedFiles: modifiedFiles,
	}, nil
}

func writeAttribute(body *hclwrite.Body, key string, val interface{}, knownResources map[string]parser.Node) {
	if val == nil {
		return
	}

	switch v := val.(type) {
	case string:
		// Check if string is a reference to another resource (e.g. aws_vpc.main.id)
		if isResourceReference(v, knownResources) {
			body.SetAttributeRaw(key, hclwrite.TokensForIdentifier(v))
		} else {
			body.SetAttributeValue(key, cty.StringVal(v))
		}
	case bool:
		body.SetAttributeValue(key, cty.BoolVal(v))
	case float64:
		// If whole number, treat as integer
		if v == float64(int64(v)) {
			body.SetAttributeValue(key, cty.NumberIntVal(int64(v)))
		} else {
			body.SetAttributeValue(key, cty.NumberFloatVal(v))
		}
	case int:
		body.SetAttributeValue(key, cty.NumberIntVal(int64(v)))
	case int64:
		body.SetAttributeValue(key, cty.NumberIntVal(v))
	case []interface{}:
		var tuple []cty.Value
		for _, elem := range v {
			if ctyVal := convertToCtyVal(elem, knownResources); !ctyVal.IsNull() {
				tuple = append(tuple, ctyVal)
			}
		}
		if len(tuple) > 0 {
			body.SetAttributeValue(key, cty.TupleVal(tuple))
		}
	case []string:
		var tuple []cty.Value
		for _, elem := range v {
			tuple = append(tuple, cty.StringVal(elem))
		}
		if len(tuple) > 0 {
			body.SetAttributeValue(key, cty.TupleVal(tuple))
		}
	case map[string]interface{}:
		objMap := make(map[string]cty.Value)
		for subKey, subVal := range v {
			if ctyVal := convertToCtyVal(subVal, knownResources); !ctyVal.IsNull() {
				objMap[subKey] = ctyVal
			}
		}
		if len(objMap) > 0 {
			body.SetAttributeValue(key, cty.ObjectVal(objMap))
		}
	}
}

func convertToCtyVal(val interface{}, knownResources map[string]parser.Node) cty.Value {
	if val == nil {
		return cty.NilVal
	}
	switch v := val.(type) {
	case string:
		return cty.StringVal(v)
	case bool:
		return cty.BoolVal(v)
	case float64:
		if v == float64(int64(v)) {
			return cty.NumberIntVal(int64(v))
		}
		return cty.NumberFloatVal(v)
	case int:
		return cty.NumberIntVal(int64(v))
	case int64:
		return cty.NumberIntVal(v)
	case []interface{}:
		var list []cty.Value
		for _, item := range v {
			if sub := convertToCtyVal(item, knownResources); !sub.IsNull() {
				list = append(list, sub)
			}
		}
		if len(list) == 0 {
			return cty.TupleVal([]cty.Value{})
		}
		return cty.TupleVal(list)
	case map[string]interface{}:
		m := make(map[string]cty.Value)
		for k, item := range v {
			if sub := convertToCtyVal(item, knownResources); !sub.IsNull() {
				m[k] = sub
			}
		}
		return cty.ObjectVal(m)
	}
	return cty.NilVal
}

func isResourceReference(s string, knownResources map[string]parser.Node) bool {
	parts := strings.Split(s, ".")
	if len(parts) >= 2 {
		candidate := fmt.Sprintf("%s.%s", parts[0], parts[1])
		if _, ok := knownResources[candidate]; ok {
			return true
		}
	}
	return false
}
