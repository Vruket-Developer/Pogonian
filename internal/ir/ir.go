package ir

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

// Application represents the complete Pogonian Intermediate Representation of a full-stack project.
type Application struct {
	Name         string          `json:"name" yaml:"name"`
	Version      string          `json:"version" yaml:"version"`
	Framework    string          `json:"framework" yaml:"framework"`
	Architecture string          `json:"architecture" yaml:"architecture"`
	Routes       []*RouteIR      `json:"routes" yaml:"routes"`
	Entities     []*EntityIR     `json:"entities" yaml:"entities"`
	Handlers     []*HandlerIR    `json:"handlers" yaml:"handlers"`
	Services     []*ServiceIR    `json:"services" yaml:"services"`
	Repositories []*RepositoryIR `json:"repositories" yaml:"repositories"`
	Database     *DatabaseIR     `json:"database,omitempty" yaml:"database,omitempty"`
	Metadata     map[string]any  `json:"metadata,omitempty" yaml:"metadata,omitempty"`
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
	Name   string    `json:"name" yaml:"name"`
	Fields []FieldIR `json:"fields" yaml:"fields"`
}

// FieldIR represents a field within an Entity.
type FieldIR struct {
	Name     string `json:"name" yaml:"name"`
	Type     string `json:"type" yaml:"type"`
	Required bool   `json:"required" yaml:"required"`
	PK       bool   `json:"pk,omitempty" yaml:"pk,omitempty"`
}

// HandlerIR represents an HTTP/Action Handler layer in the backend.
type HandlerIR struct {
	Name            string   `json:"name" yaml:"name"`
	Package         string   `json:"package" yaml:"package"`
	File            string   `json:"file" yaml:"file"`
	CallsServices   []string `json:"calls_services" yaml:"calls_services"`
	DirectDBAccess  bool     `json:"direct_db_access" yaml:"direct_db_access"`
	HandledMethods  []string `json:"handled_methods" yaml:"handled_methods"`
}

// ServiceIR represents a Business Logic layer in the backend.
type ServiceIR struct {
	Name       string   `json:"name" yaml:"name"`
	Package    string   `json:"package" yaml:"package"`
	File       string   `json:"file" yaml:"file"`
	CallsRepos []string `json:"calls_repos" yaml:"calls_repos"`
	Methods    []string `json:"methods" yaml:"methods"`
}

// RepositoryIR represents a Data Access layer in the backend.
type RepositoryIR struct {
	Name        string   `json:"name" yaml:"name"`
	Package     string   `json:"package" yaml:"package"`
	File        string   `json:"file" yaml:"file"`
	TargetTable string   `json:"target_table" yaml:"target_table"`
	Methods     []string `json:"methods" yaml:"methods"`
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

// ToJSON serializes the Pogonian IR application to JSON.
func (app *Application) ToJSON() ([]byte, error) {
	return json.MarshalIndent(app, "", "  ")
}

// ToYAML serializes the Pogonian IR application to YAML.
func (app *Application) ToYAML() ([]byte, error) {
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
