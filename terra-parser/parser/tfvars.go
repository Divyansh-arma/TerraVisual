package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

var varRefRegex = regexp.MustCompile(`\$\{?var\.([a-zA-Z0-9_-]+)\}?`)

// CollectAndParseTFVars searches the root directory and subdirectories for terraform.tfvars,
// *.auto.tfvars, and default values in variables.tf to resolve variable references.
func CollectAndParseTFVars(rootDir string) (map[string]interface{}, error) {
	vars := make(map[string]interface{})

	// 1. First scan variables.tf to extract default variable values
	_ = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		name := info.Name()
		if name == "variables.tf" || strings.HasSuffix(name, ".tf") {
			extractDefaultsFromVariablesTF(path, vars)
		}
		return nil
	})

	// 2. Scan and parse .tfvars files (which override defaults)
	_ = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}

		name := info.Name()
		if name == "terraform.tfvars" || strings.HasSuffix(name, ".auto.tfvars") || strings.HasSuffix(name, ".tfvars") {
			parseTFVarsFile(path, vars)
		}
		return nil
	})

	return vars, nil
}

func parseTFVarsFile(filePath string, vars map[string]interface{}) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return
	}

	file, diags := hclsyntax.ParseConfig(src, filePath, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() || file == nil || file.Body == nil {
		return
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok || body == nil {
		return
	}

	for attrName, attr := range body.Attributes {
		vars[attrName] = extractHCLAttributeValue(attr.Expr)
	}
}

func extractDefaultsFromVariablesTF(filePath string, vars map[string]interface{}) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return
	}

	file, diags := hclsyntax.ParseConfig(src, filePath, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() || file == nil || file.Body == nil {
		return
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok || body == nil {
		return
	}

	for _, block := range body.Blocks {
		if block.Type == "variable" && len(block.Labels) == 1 {
			varName := block.Labels[0]
			if _, alreadySet := vars[varName]; !alreadySet && block.Body != nil {
				if defaultAttr, exists := block.Body.Attributes["default"]; exists {
					vars[varName] = extractHCLAttributeValue(defaultAttr.Expr)
				}
			}
		}
	}
}

// ResolveVarValue evaluates variable references (e.g. "var.vpc_cidr" or "${var.name}-cluster")
// against the parsed tfvars lookup table.
func ResolveVarValue(val interface{}, vars map[string]interface{}) interface{} {
	if val == nil || vars == nil {
		return val
	}

	switch v := val.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		// Exact match: "var.vpc_cidr" or "${var.vpc_cidr}"
		if strings.HasPrefix(trimmed, "var.") {
			varName := strings.TrimPrefix(trimmed, "var.")
			if resolved, exists := vars[varName]; exists {
				return resolved
			}
		} else if strings.HasPrefix(trimmed, "${var.") && strings.HasSuffix(trimmed, "}") {
			varName := strings.TrimSuffix(strings.TrimPrefix(trimmed, "${var."), "}")
			if resolved, exists := vars[varName]; exists {
				return resolved
			}
		}

		// String interpolation match e.g. "prefix-${var.name}-suffix"
		if varRefRegex.MatchString(v) {
			resolvedStr := varRefRegex.ReplaceAllStringFunc(v, func(match string) string {
				sub := varRefRegex.FindStringSubmatch(match)
				if len(sub) > 1 {
					varName := sub[1]
					if resolved, exists := vars[varName]; exists {
						return fmt.Sprintf("%v", resolved)
					}
				}
				return match
			})
			return resolvedStr
		}
		return v

	case []interface{}:
		resolvedSlice := make([]interface{}, len(v))
		for i, elem := range v {
			resolvedSlice[i] = ResolveVarValue(elem, vars)
		}
		return resolvedSlice

	case []string:
		resolvedSlice := make([]string, len(v))
		for i, elem := range v {
			resolvedVal := ResolveVarValue(elem, vars)
			if strVal, ok := resolvedVal.(string); ok {
				resolvedSlice[i] = strVal
			} else {
				resolvedSlice[i] = fmt.Sprintf("%v", resolvedVal)
			}
		}
		return resolvedSlice

	case map[string]interface{}:
		resolvedMap := make(map[string]interface{}, len(v))
		for k, elem := range v {
			resolvedMap[k] = ResolveVarValue(elem, vars)
		}
		return resolvedMap
	}

	return val
}

// ResolveAttributesWithVars recursively resolves all attributes in a map with tfvars.
func ResolveAttributesWithVars(attrs map[string]interface{}, vars map[string]interface{}) map[string]interface{} {
	if attrs == nil || vars == nil {
		return attrs
	}

	resolved := make(map[string]interface{}, len(attrs))
	for k, v := range attrs {
		resolved[k] = ResolveVarValue(v, vars)
	}
	return resolved
}
