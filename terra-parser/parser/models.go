package parser

// TFState represents the top-level Terraform v4 state format.
type TFState struct {
	Version          int          `json:"version"`
	TerraformVersion string       `json:"terraform_version"`
	Serial           int          `json:"serial"`
	Lineage          string       `json:"lineage"`
	Resources        []TFResource `json:"resources"`
}

// TFResource represents a resource entry in the Terraform state file.
type TFResource struct {
	Module       string       `json:"module,omitempty"`
	Mode         string       `json:"mode"`
	Type         string       `json:"type"`
	Name         string       `json:"name"`
	Provider     string       `json:"provider"`
	Instances    []TFInstance `json:"instances"`
	Dependencies []string     `json:"dependencies,omitempty"`
}

// TFInstance represents a single instance of a resource in the state file.
type TFInstance struct {
	SchemaVersion int                    `json:"schema_version,omitempty"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
	Dependencies  []string               `json:"dependencies,omitempty"`
}

// Position represents 2D canvas coordinates for React Flow.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// SecurityIssue represents a security misconfiguration found in infrastructure code.
type SecurityIssue struct {
	RuleID      string `json:"ruleId"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// AttributeDiff represents a specific attribute divergence between Terraform state and code.
type AttributeDiff struct {
	Field      string      `json:"field"`
	StateValue interface{} `json:"stateValue"`
	CodeValue  interface{} `json:"codeValue"`
}

// NodeData contains metadata specific to the infrastructure resource.
type NodeData struct {
	Label          string                 `json:"label"`
	Provider       string                 `json:"provider"`
	ResourceType   string                 `json:"resourceType"`
	Module         string                 `json:"module"`
	IsDataSource   bool                   `json:"isDataSource"`
	IsContainer    bool                   `json:"isContainer,omitempty"`
	DriftStatus    string                 `json:"driftStatus"`
	Attributes     map[string]interface{} `json:"attributes,omitempty"`
	SecurityIssues []SecurityIssue        `json:"securityIssues,omitempty"`
	DriftDiffs     []AttributeDiff        `json:"driftDiffs,omitempty"`
}

// Node represents a React Flow node for canvas rendering.
type Node struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Position Position `json:"position"`
	Data     NodeData `json:"data"`
	ParentID string   `json:"parentId,omitempty"`
}

// Edge represents a React Flow directed edge connecting dependencies.
type Edge struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	Type     string `json:"type"`
	Animated bool   `json:"animated"`
}

// GraphResponse is the standardized output format for the visual graph canvas.
type GraphResponse struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}
