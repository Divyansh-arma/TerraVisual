package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

type blockContext struct {
	nodeID     string
	blockType  string // "resource" or "module"
	moduleName string
	block      *hclsyntax.Block
}

// ParseHCLDirectory recursively reads all .tf files in the specified directory and its subdirectories,
// supporting enterprise modular Terraform setups, first-class module blocks, and robust structural AST parsing.
func ParseHCLDirectory(dirPath string) (*GraphResponse, error) {
	absRootDir, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", dirPath, err)
	}

	// 1. Discover all .tf files recursively, skipping ignored directories
	tfFilesByDir, dirErr := collectTFFilesRecursively(absRootDir)
	if dirErr != nil {
		return nil, fmt.Errorf("failed to scan directories in %s: %w", dirPath, dirErr)
	}

	// 1.5. Ingest terraform.tfvars and default variable values
	tfVars, _ := CollectAndParseTFVars(absRootDir)

	if len(tfFilesByDir) == 0 {
		return &GraphResponse{
			Nodes: []Node{},
			Edges: []Edge{},
		}, nil
	}

	nodeMap := make(map[string]Node)
	knownNodes := make(map[string]bool)
	moduleSourceMap := make(map[string]string) // local dir path -> "module.<name>"
	var allBlockContexts []blockContext

	// Pass 1A: Parse root directory files first to discover top-level module blocks & local source paths
	if rootFiles, exists := tfFilesByDir[absRootDir]; exists {
		for _, tfFilePath := range rootFiles {
			body, parseErr := parseHCLFile(tfFilePath)
			if parseErr != nil {
				continue // Graceful recovery: skip unparseable files without failing whole job
			}

			for _, block := range body.Blocks {
				if block.Type == "module" && len(block.Labels) == 1 {
					modName := block.Labels[0]
					nodeID := fmt.Sprintf("module.%s", modName)
					attributes := extractBlockAttributes(block.Body)

					// Detect local directory source path (e.g. ./modules/vpc or ./Modules/EKS)
					if srcVal, ok := attributes["source"].(string); ok && srcVal != "" {
						if strings.HasPrefix(srcVal, ".") || strings.HasPrefix(srcVal, "/") {
							absSourcePath := filepath.Clean(filepath.Join(absRootDir, srcVal))
							moduleSourceMap[absSourcePath] = nodeID
						}
					}

					node := Node{
						ID:   nodeID,
						Type: "infrastructureNode",
						Position: Position{
							X: 0,
							Y: 0,
						},
						Data: NodeData{
							Label:        fmt.Sprintf("module.%s", modName),
							Provider:     "terraform",
							ResourceType: "module",
							Module:       "root",
							IsDataSource: false,
							DriftStatus:  "unknown",
							Attributes:   attributes,
						},
					}

					nodeMap[nodeID] = node
					knownNodes[nodeID] = true
					allBlockContexts = append(allBlockContexts, blockContext{
						nodeID:     nodeID,
						blockType:  "module",
						moduleName: "root",
						block:      block,
					})
				}
			}
		}
	}

	// Pass 1B: Parse all directories and files (root and submodules)
	// Sort directories deterministically
	sortedDirs := make([]string, 0, len(tfFilesByDir))
	for dir := range tfFilesByDir {
		sortedDirs = append(sortedDirs, dir)
	}
	sort.Strings(sortedDirs)

	for _, currentDir := range sortedDirs {
		tfFiles := tfFilesByDir[currentDir]
		sort.Strings(tfFiles)

		// Determine module context for this directory
		currentModule := "root"
		parentModuleID := ""
		if currentDir != absRootDir {
			if modID, matched := matchModuleSource(currentDir, moduleSourceMap); matched {
				currentModule = modID
				parentModuleID = modID
			} else {
				relDir, _ := filepath.Rel(absRootDir, currentDir)
				currentModule = filepath.ToSlash(relDir)
			}
		}

		for _, tfFilePath := range tfFiles {
			body, parseErr := parseHCLFile(tfFilePath)
			if parseErr != nil {
				continue
			}

			for _, block := range body.Blocks {
				if block.Type == "resource" && len(block.Labels) == 2 {
					resType := block.Labels[0]
					resName := block.Labels[1]

					var nodeID string
					if currentModule == "root" {
						nodeID = fmt.Sprintf("%s.%s", resType, resName)
					} else {
						nodeID = fmt.Sprintf("%s.%s.%s", currentModule, resType, resName)
					}

					provider := inferProviderFromResourceType(resType)
					attributes := extractBlockAttributes(block.Body)

					// Determine parent ID:
					// If resource has explicit vpc_id referencing a VPC, that VPC takes priority.
					// Otherwise, if inside a child module, parentID is the parent module container.
					parentID := parentModuleID
					if vpcIDVal, ok := attributes["vpc_id"].(string); ok && vpcIDVal != "" {
						if strings.HasPrefix(vpcIDVal, "aws_vpc.") {
							parts := strings.Split(vpcIDVal, ".")
							if len(parts) >= 2 {
								parentID = fmt.Sprintf("%s.%s", parts[0], parts[1])
							}
						} else if strings.Contains(vpcIDVal, "aws_vpc.") {
							// e.g. module.vpc.aws_vpc.main.id
							parentID = cleanResourceReference(vpcIDVal)
						} else if vpcIDVal != "" && !strings.Contains(vpcIDVal, "(") {
							parentID = vpcIDVal
						}
					}

					node := Node{
						ID:   nodeID,
						Type: "infrastructureNode",
						Position: Position{
							X: 0,
							Y: 0,
						},
						Data: NodeData{
							Label:        resName,
							Provider:     provider,
							ResourceType: resType,
							Module:       currentModule,
							IsDataSource: false,
							DriftStatus:  "unknown",
							Attributes:   attributes,
						},
						ParentID: parentID,
					}

					nodeMap[nodeID] = node
					knownNodes[nodeID] = true
					// Also register short unqualified ID for reference matching if unique
					shortID := fmt.Sprintf("%s.%s", resType, resName)
					knownNodes[shortID] = true

					allBlockContexts = append(allBlockContexts, blockContext{
						nodeID:     nodeID,
						blockType:  "resource",
						moduleName: currentModule,
						block:      block,
					})
				} else if block.Type == "module" && len(block.Labels) == 1 && currentDir != absRootDir {
					// Submodule within a submodule
					modName := block.Labels[0]
					// Flatten wrapper: avoid creating duplicate wrapper if name matches parent module
					if modName == strings.TrimPrefix(currentModule, "module.") {
						continue
					}

					nodeID := fmt.Sprintf("%s.module.%s", currentModule, modName)
					attributes := extractBlockAttributes(block.Body)

					node := Node{
						ID:   nodeID,
						Type: "infrastructureNode",
						Position: Position{
							X: 0,
							Y: 0,
						},
						Data: NodeData{
							Label:        fmt.Sprintf("module.%s", modName),
							Provider:     "terraform",
							ResourceType: "module",
							Module:       currentModule,
							IsDataSource: false,
							DriftStatus:  "unknown",
							Attributes:   attributes,
						},
						ParentID: parentModuleID,
					}

					nodeMap[nodeID] = node
					knownNodes[nodeID] = true
					allBlockContexts = append(allBlockContexts, blockContext{
						nodeID:     nodeID,
						blockType:  "module",
						moduleName: currentModule,
						block:      block,
					})
				}
			}
		}
	}

	// Pass 2: Inter-Module & Resource Dependency Detection
	edgeMap := make(map[string]Edge)

	for _, bc := range allBlockContexts {
		sourceID := bc.nodeID

		walker := &astWalker{
			onEnter: func(node hclsyntax.Node) {
				if expr, ok := node.(hclsyntax.Expression); ok {
					for _, traversal := range expr.Variables() {
						targetID := resolveReferenceWithModules(traversal, knownNodes, bc.moduleName, nodeMap)
						if targetID != "" && targetID != sourceID {
							edgeID := fmt.Sprintf("e-%s-%s", sourceID, targetID)
							if _, exists := edgeMap[edgeID]; !exists {
								edgeMap[edgeID] = Edge{
									ID:       edgeID,
									Source:   sourceID,
									Target:   targetID,
									Type:     "smoothstep",
									Animated: true,
								}
							}
						}
					}
				}
			},
		}

		if bc.block.Body != nil {
			_ = hclsyntax.Walk(bc.block.Body, walker)
		}
	}

	// Unroll modular architectures into visual VPC containers and subnets
	nodeMap, edgeMap = UnrollModularArchitecture(nodeMap, edgeMap, tfVars)

	// Apply AWS Architecture Hierarchy Topology (VPC -> AZ -> Subnet -> Resource)
	nodeMap, edgeMap = ApplyAWSTopology(nodeMap, edgeMap)

	nodes := make([]Node, 0, len(nodeMap))
	for _, n := range nodeMap {
		nodes = append(nodes, n)
	}

	edges := make([]Edge, 0, len(edgeMap))
	for _, e := range edgeMap {
		edges = append(edges, e)
	}

	// Sort nodes topologically with parents before children
	SortNodesParentsFirst(nodes)

	sort.Slice(edges, func(i, j int) bool {
		return edges[i].ID < edges[j].ID
	})

	return &GraphResponse{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// collectTFFilesRecursively walks dirPath and groups .tf files by directory, skipping ignored folders.
func collectTFFilesRecursively(rootDir string) (map[string][]string, error) {
	filesByDir := make(map[string][]string)
	visitedDirs := make(map[string]bool)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil // Skip unreadable paths without terminating walk
		}

		if info.IsDir() {
			dirName := info.Name()
			// Skip hidden and dependency cache directories
			if strings.HasPrefix(dirName, ".") ||
				dirName == "node_modules" ||
				dirName == "dist" ||
				dirName == "bin" ||
				dirName == "target" ||
				dirName == ".terraform" ||
				dirName == ".terragrunt-cache" {
				return filepath.SkipDir
			}

			// Prevent infinite symlink loops
			realPath, err := filepath.EvalSymlinks(path)
			if err == nil {
				if visitedDirs[realPath] {
					return filepath.SkipDir
				}
				visitedDirs[realPath] = true
			}
			return nil
		}

		if strings.HasSuffix(info.Name(), ".tf") && !strings.HasSuffix(info.Name(), ".tf.json") {
			dir := filepath.Dir(path)
			filesByDir[dir] = append(filesByDir[dir], path)
		}

		return nil
	})

	return filesByDir, err
}

func parseHCLFile(filePath string) (*hclsyntax.Body, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	file, diags := hclsyntax.ParseConfig(src, filePath, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, diags
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("file body is not *hclsyntax.Body")
	}

	return body, nil
}

func matchModuleSource(dirPath string, sourceMap map[string]string) (string, bool) {
	cleanDir := filepath.Clean(dirPath)
	if modID, exists := sourceMap[cleanDir]; exists {
		return modID, true
	}

	for srcPath, modID := range sourceMap {
		if strings.EqualFold(srcPath, cleanDir) {
			return modID, true
		}
		rel, err := filepath.Rel(srcPath, cleanDir)
		if err == nil && !strings.HasPrefix(rel, "..") {
			return modID, true
		}
		relFold, errFold := filepath.Rel(strings.ToLower(srcPath), strings.ToLower(cleanDir))
		if errFold == nil && !strings.HasPrefix(relFold, "..") {
			return modID, true
		}
	}

	return "", false
}

func extractBlockAttributes(body *hclsyntax.Body) map[string]interface{} {
	attributes := make(map[string]interface{})
	if body == nil {
		return attributes
	}

	for attrName, attr := range body.Attributes {
		if attrName == "depends_on" {
			continue
		}
		attributes[attrName] = extractHCLAttributeValue(attr.Expr)
	}

	return attributes
}

// resolveReferenceWithModules resolves HCL traversals to matching resources or modules.
// Handles module references (e.g. module.vpc.vpc_id -> module.vpc), resource references,
// and scoped module child references.
func resolveReferenceWithModules(
	traversal hcl.Traversal,
	knownNodes map[string]bool,
	currentModule string,
	nodeMap map[string]Node,
) string {
	if len(traversal) < 1 {
		return ""
	}

	rootName := traversal.RootName()

	// 1. Module reference: module.<name>.<output>
	if rootName == "module" && len(traversal) >= 2 {
		if modAttr, ok := traversal[1].(hcl.TraverseAttr); ok {
			modID := fmt.Sprintf("module.%s", modAttr.Name)
			if knownNodes[modID] {
				return modID
			}
			// Check scoped module e.g. module.app.module.vpc
			if currentModule != "" && currentModule != "root" {
				scopedModID := fmt.Sprintf("%s.module.%s", currentModule, modAttr.Name)
				if knownNodes[scopedModID] {
					return scopedModID
				}
			}
			return modID
		}
	}

	// 2. Resource reference: <resType>.<resName>.<attr>
	if len(traversal) >= 2 {
		if attr, ok := traversal[1].(hcl.TraverseAttr); ok {
			// Check scoped node first if inside module
			if currentModule != "" && currentModule != "root" {
				scopedCandidate := fmt.Sprintf("%s.%s.%s", currentModule, rootName, attr.Name)
				if knownNodes[scopedCandidate] {
					return scopedCandidate
				}
			}

			// Check global / root candidate
			candidate := fmt.Sprintf("%s.%s", rootName, attr.Name)
			if knownNodes[candidate] {
				return candidate
			}

			// Check if candidate matches any node's short name in nodeMap
			for id, n := range nodeMap {
				if n.Data.ResourceType == rootName && n.Data.Label == attr.Name {
					return id
				}
			}
		}
	}

	return ""
}

func cleanResourceReference(raw string) string {
	raw = strings.TrimSpace(raw)
	parts := strings.Split(raw, ".")
	for i := 0; i < len(parts)-1; i++ {
		if strings.HasPrefix(parts[i], "aws_") || strings.HasPrefix(parts[i], "azurerm_") || strings.HasPrefix(parts[i], "google_") {
			return fmt.Sprintf("%s.%s", parts[i], parts[i+1])
		}
	}
	return raw
}

// extractHCLAttributeValue performs pure structural AST inspection of any HCL expression.
// It never evaluates runtime contexts or calls functions dynamically, guaranteeing zero crashes.
func extractHCLAttributeValue(expr hclsyntax.Expression) interface{} {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		val := e.Val
		if !val.IsKnown() || val.IsNull() {
			return nil
		}
		switch val.Type() {
		case cty.String:
			return val.AsString()
		case cty.Bool:
			return val.True()
		case cty.Number:
			bf := val.AsBigFloat()
			f, _ := bf.Float64()
			return f
		default:
			return val.AsString()
		}

	case *hclsyntax.TemplateWrapExpr:
		return extractHCLAttributeValue(e.Wrapped)

	case *hclsyntax.TemplateExpr:
		if e.IsStringLiteral() {
			var sb strings.Builder
			for _, part := range e.Parts {
				if lit, ok := part.(*hclsyntax.LiteralValueExpr); ok && lit.Val.Type() == cty.String {
					sb.WriteString(lit.Val.AsString())
				}
			}
			return sb.String()
		}
		// String template with interpolations: extract as readable composite string
		var sb strings.Builder
		for _, part := range e.Parts {
			switch p := part.(type) {
			case *hclsyntax.LiteralValueExpr:
				if p.Val.Type() == cty.String {
					sb.WriteString(p.Val.AsString())
				}
			case *hclsyntax.ScopeTraversalExpr:
				sb.WriteString("${" + formatScopeTraversal(p.Traversal) + "}")
			default:
				sb.WriteString(fmt.Sprintf("${%v}", extractHCLAttributeValue(p)))
			}
		}
		return sb.String()

	case *hclsyntax.ScopeTraversalExpr:
		return formatScopeTraversal(e.Traversal)

	case *hclsyntax.TupleConsExpr:
		list := make([]interface{}, len(e.Exprs))
		for i, elem := range e.Exprs {
			list[i] = extractHCLAttributeValue(elem)
		}
		return list

	case *hclsyntax.ObjectConsKeyExpr:
		val, diags := e.Value(nil)
		if !diags.HasErrors() && val.IsKnown() && !val.IsNull() && val.Type() == cty.String {
			return val.AsString()
		}
		if e.Wrapped != nil {
			return extractHCLAttributeValue(e.Wrapped)
		}
		return "key"

	case *hclsyntax.ObjectConsExpr:
		obj := make(map[string]interface{})
		for _, item := range e.Items {
			var key string
			val, diags := item.KeyExpr.Value(nil)
			if !diags.HasErrors() && val.IsKnown() && !val.IsNull() && val.Type() == cty.String {
				key = val.AsString()
			} else {
				key = fmt.Sprintf("%v", extractHCLAttributeValue(item.KeyExpr))
			}
			obj[key] = extractHCLAttributeValue(item.ValueExpr)
		}
		return obj

	case *hclsyntax.FunctionCallExpr:
		// Structural function call e.g. slice(var.subnets, 0, 2), length(var.azs), lookup(local.tags, "Env")
		argStrs := make([]string, len(e.Args))
		for i, arg := range e.Args {
			argStrs[i] = fmt.Sprintf("%v", extractHCLAttributeValue(arg))
		}
		return fmt.Sprintf("%s(%s)", e.Name, strings.Join(argStrs, ", "))

	case *hclsyntax.ConditionalExpr:
		// Ternary conditional e.g. var.enabled ? 1 : 0
		condStr := fmt.Sprintf("%v", extractHCLAttributeValue(e.Condition))
		trueStr := fmt.Sprintf("%v", extractHCLAttributeValue(e.TrueResult))
		falseStr := fmt.Sprintf("%v", extractHCLAttributeValue(e.FalseResult))
		return fmt.Sprintf("%s ? %s : %s", condStr, trueStr, falseStr)

	case *hclsyntax.SplatExpr:
		// Splat expression e.g. aws_subnet.public[*].id
		sourceStr := fmt.Sprintf("%v", extractHCLAttributeValue(e.Source))
		eachStr := fmt.Sprintf("%v", extractHCLAttributeValue(e.Each))
		return fmt.Sprintf("%s[*].%s", sourceStr, eachStr)

	case *hclsyntax.BinaryOpExpr:
		lhs := fmt.Sprintf("%v", extractHCLAttributeValue(e.LHS))
		rhs := fmt.Sprintf("%v", extractHCLAttributeValue(e.RHS))
		return fmt.Sprintf("%s %s %s", lhs, e.Op.Type.GoString(), rhs)

	case *hclsyntax.UnaryOpExpr:
		val := fmt.Sprintf("%v", extractHCLAttributeValue(e.Val))
		return fmt.Sprintf("%s%s", e.Op.Type.GoString(), val)

	case *hclsyntax.ForExpr:
		coll := fmt.Sprintf("%v", extractHCLAttributeValue(e.CollExpr))
		val := fmt.Sprintf("%v", extractHCLAttributeValue(e.ValExpr))
		return fmt.Sprintf("[for in %s : %s]", coll, val)
	}

	return nil
}

func formatScopeTraversal(traversal hcl.Traversal) string {
	var parts []string
	for _, traverser := range traversal {
		switch t := traverser.(type) {
		case hcl.TraverseRoot:
			parts = append(parts, t.Name)
		case hcl.TraverseAttr:
			parts = append(parts, t.Name)
		case hcl.TraverseIndex:
			parts = append(parts, fmt.Sprintf("%v", t.Key))
		}
	}
	return strings.Join(parts, ".")
}

// inferProviderFromResourceType extracts the provider name from the resource type prefix (e.g. "aws_vpc" -> "aws").
func inferProviderFromResourceType(resType string) string {
	parts := strings.Split(resType, "_")
	if len(parts) > 0 && parts[0] != "" {
		return strings.ToLower(parts[0])
	}
	return "unknown"
}

type astWalker struct {
	onEnter func(node hclsyntax.Node)
}

func (w *astWalker) Enter(node hclsyntax.Node) hcl.Diagnostics {
	if w.onEnter != nil {
		w.onEnter(node)
	}
	return nil
}

func (w *astWalker) Exit(node hclsyntax.Node) hcl.Diagnostics {
	return nil
}
