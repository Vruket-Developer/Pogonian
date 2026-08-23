package ir

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

// SchemaVersion defines the current Pogonian IR schema version.
const SchemaVersion = "1.0.0"

// Application represents the complete Pogonian Intermediate Representation of a full-stack project.
type Application struct {
	SchemaVersion string          `json:"schema_version" yaml:"schema_version"`
	Name          string          `json:"name" yaml:"name"`
	Version       string          `json:"version" yaml:"version"`
	Framework     string          `json:"framework" yaml:"framework"`
	Architecture  string          `json:"architecture" yaml:"architecture"`
	Routes        []*RouteIR      `json:"routes" yaml:"routes"`
	Entities      []*EntityIR     `json:"entities" yaml:"entities"`
	Handlers      []*HandlerIR    `json:"handlers" yaml:"handlers"`
	Services      []*ServiceIR    `json:"services" yaml:"services"`
	Repositories  []*RepositoryIR `json:"repositories" yaml:"repositories"`
	Database      *DatabaseIR     `json:"database,omitempty" yaml:"database,omitempty"`
	Contracts     []*ContractIR   `json:"contracts,omitempty" yaml:"contracts,omitempty"`
	Features      []*FeatureIR    `json:"features,omitempty" yaml:"features,omitempty"`
	Metadata      map[string]any  `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// RouteIR represents an endpoint or page in the application.
type RouteIR struct {
	Path        string   `json:"path" yaml:"path"`
	Methods     []string `json:"methods" yaml:"methods"`
	PagePath    string   `json:"page_path,omitempty" yaml:"page_path,omitempty"`
	LayoutChain []string `json:"layout_chain,omitempty" yaml:"layout_chain,omitempty"`
	Params      []string `json:"params,omitempty" yaml:"params,omitempty"`
	Handler     string   `json:"handler,omitempty" yaml:"handler,omitempty"`
}

// EntityIR represents a domain model.
type EntityIR struct {
	Name      string    `json:"name" yaml:"name"`
	Package   string    `json:"package,omitempty" yaml:"package,omitempty"`
	File      string    `json:"file,omitempty" yaml:"file,omitempty"`
	Fields    []FieldIR `json:"fields" yaml:"fields"`
	TableName string    `json:"table_name,omitempty" yaml:"table_name,omitempty"`
}

// FieldIR represents a field within an Entity.
type FieldIR struct {
	Name     string `json:"name" yaml:"name"`
	Type     string `json:"type" yaml:"type"`
	Required bool   `json:"required" yaml:"required"`
	PK       bool   `json:"pk,omitempty" yaml:"pk,omitempty"`
	JSONTag  string `json:"json_tag,omitempty" yaml:"json_tag,omitempty"`
}

// MethodIR represents a Go method signature.
type MethodIR struct {
	Name       string   `json:"name" yaml:"name"`
	Parameters []string `json:"parameters" yaml:"parameters"`
	Returns    []string `json:"returns" yaml:"returns"`
}

// HandlerIR represents an HTTP/Action Handler layer in the backend.
type HandlerIR struct {
	Name           string      `json:"name" yaml:"name"`
	Package        string      `json:"package" yaml:"package"`
	File           string      `json:"file" yaml:"file"`
	CallsServices  []string    `json:"calls_services" yaml:"calls_services"`
	DirectDBAccess bool        `json:"direct_db_access" yaml:"direct_db_access"`
	HandledMethods []string    `json:"handled_methods" yaml:"handled_methods"`
	Methods        []*MethodIR `json:"methods,omitempty" yaml:"methods,omitempty"`
}

// ServiceIR represents a Business Logic layer in the backend.
type ServiceIR struct {
	Name       string      `json:"name" yaml:"name"`
	Package    string      `json:"package" yaml:"package"`
	File       string      `json:"file" yaml:"file"`
	CallsRepos []string    `json:"calls_repos" yaml:"calls_repos"`
	Methods    []*MethodIR `json:"methods,omitempty" yaml:"methods,omitempty"`
}

// RepositoryIR represents a Data Access layer in the backend.
type RepositoryIR struct {
	Name        string      `json:"name" yaml:"name"`
	Package     string      `json:"package" yaml:"package"`
	File        string      `json:"file" yaml:"file"`
	TargetTable string      `json:"target_table" yaml:"target_table"`
	Methods     []*MethodIR `json:"methods,omitempty" yaml:"methods,omitempty"`
}

// DatabaseIR represents database configurations and schemas.
type DatabaseIR struct {
	Driver string    `json:"driver" yaml:"driver"`
	Tables []TableIR `json:"tables" yaml:"tables"`
}

// TableIR represents a database table schema.
type TableIR struct {
	Name    string   `json:"name" yaml:"name"`
	Columns []string `json:"columns" yaml:"columns"`
}

// ContractIR represents an architectural rule or boundary guarantee.
type ContractIR struct {
	Name        string `json:"name" yaml:"name"`
	SourceLayer string `json:"source_layer" yaml:"source_layer"`
	TargetLayer string `json:"target_layer" yaml:"target_layer"`
	Rule        string `json:"rule" yaml:"rule"` // "ALLOW", "DENY"
	Enforced    bool   `json:"enforced" yaml:"enforced"`
}

// FeatureIR represents a cohesive full-stack module.
type FeatureIR struct {
	Name         string   `json:"name" yaml:"name"`
	Route        string   `json:"route" yaml:"route"`
	Entity       string   `json:"entity" yaml:"entity"`
	Handler      string   `json:"handler" yaml:"handler"`
	Service      string   `json:"service" yaml:"service"`
	Repository   string   `json:"repository" yaml:"repository"`
	Capabilities []string `json:"capabilities" yaml:"capabilities"`
}

// ToJSON serializes the Pogonian IR application to JSON.
func (app *Application) ToJSON() ([]byte, error) {
	if app.SchemaVersion == "" {
		app.SchemaVersion = SchemaVersion
	}
	return json.MarshalIndent(app, "", "  ")
}

// ToYAML serializes the Pogonian IR application to YAML.
func (app *Application) ToYAML() ([]byte, error) {
	if app.SchemaVersion == "" {
		app.SchemaVersion = SchemaVersion
	}
	return yaml.Marshal(app)
}

// FromJSON deserializes JSON data into a Pogonian IR application.
func FromJSON(data []byte) (*Application, error) {
	var app Application
	if err := json.Unmarshal(data, &app); err != nil {
		return nil, err
	}
	return &app, nil
}
