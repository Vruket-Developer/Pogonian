package compiler

import (
	"fmt"
	"path/filepath"
	"strings"

	"pogonian/internal/architecture"
	"pogonian/pkg/capability"
	"pogonian/pkg/config"
	"pogonian/pkg/ir"
)

// CompilationResult captures the end-to-end multi-target compilation output.
type CompilationResult struct {
	AppName      string                                    `json:"app_name"`
	AppIR        *ir.Application                           `json:"app_ir"`
	Artifacts    map[ir.TargetPlatform]*LoweredArtifact    `json:"artifacts"`
	Warnings     []string                                  `json:"warnings"`
	CapabilityOK bool                                      `json:"capability_ok"`
}

// Pipeline executes the multi-stage compiler: Parse ➔ Analyze ➔ Validate ➔ Optimize ➔ Lower.
type Pipeline struct {
	ProjectDir    string
	Config        *config.Config
	capValidator  *capability.Validator
	archValidator *architecture.Validator
}

// NewPipeline creates a new universal compilation pipeline.
func NewPipeline(projectDir string) (*Pipeline, error) {
	cfg, err := config.Load(projectDir)
	if err != nil {
		cfg = config.DefaultConfig(filepath.Base(projectDir))
	}

	return &Pipeline{
		ProjectDir:    projectDir,
		Config:        cfg,
		capValidator:  capability.NewValidator(),
		archValidator: architecture.NewValidator(false),
	}, nil
}

// Compile executes the compilation pipeline for specified target platforms.
func (p *Pipeline) Compile(targets []ir.TargetPlatform) (*CompilationResult, error) {
	if len(targets) == 0 {
		targets = []ir.TargetPlatform{ir.TargetWeb}
	}

	// ---------------------------------------------------------------------
	// Stage 1: Source Parsing & AST Extraction
	// ---------------------------------------------------------------------
	analyzer, err := architecture.NewAnalyzer(p.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create architecture analyzer: %w", err)
	}

	appIR, graph, err := analyzer.Analyze()
	if err != nil {
		return nil, fmt.Errorf("architecture analysis failed: %w", err)
	}
	appIR.SupportedTargets = targets

	// ---------------------------------------------------------------------
	// Stage 2: Semantic Analysis
	// ---------------------------------------------------------------------
	if err := p.performSemanticAnalysis(appIR); err != nil {
		return nil, fmt.Errorf("semantic analysis failed: %w", err)
	}

	// ---------------------------------------------------------------------
	// Stage 3: Architectural & Capability Validation
	// ---------------------------------------------------------------------
	var warnings []string

	// 3a. Architectural 5-Tier Boundary Enforcements
	archViolations := p.archValidator.Validate(appIR, graph)
	for _, v := range archViolations {
		if v.Severity == "ERROR" {
			return nil, fmt.Errorf("architecture boundary violation in %s: %s", v.Component, v.Message)
		}
		warnings = append(warnings, fmt.Sprintf("[%s] %s: %s", v.Severity, v.Component, v.Message))
	}

	// 3b. Target Device Capability Constraints
	capViolations := p.capValidator.CheckCompatibility(appIR, targets)
	for _, v := range capViolations {
		if v.Severity == "ERROR" {
			return nil, fmt.Errorf("capability check failed on platform '%s': %s (Recommendation: %s)",
				v.Platform, v.Message, v.Recommendation)
		}
		warnings = append(warnings, fmt.Sprintf("Capability Warning [%s]: %s", v.Platform, v.Message))
	}

	// ---------------------------------------------------------------------
	// Stage 4: Optimization
	// ---------------------------------------------------------------------
	p.optimizeIR(appIR)

	// ---------------------------------------------------------------------
	// Stage 5: Platform Lowering & Code Generation
	// ---------------------------------------------------------------------
	artifacts := make(map[ir.TargetPlatform]*LoweredArtifact)
	for _, target := range targets {
		lowerer, err := GetLowerer(target)
		if err != nil {
			return nil, fmt.Errorf("target platform '%s' not supported by compiler: %w", target, err)
		}

		artifact, err := lowerer.Lower(appIR, p.ProjectDir)
		if err != nil {
			return nil, fmt.Errorf("lowering failed for target '%s': %w", target, err)
		}
		artifacts[target] = artifact
	}

	return &CompilationResult{
		AppName:      appIR.Name,
		AppIR:        appIR,
		Artifacts:    artifacts,
		Warnings:     warnings,
		CapabilityOK: true,
	}, nil
}

func (p *Pipeline) performSemanticAnalysis(app *ir.Application) error {
	routeMap := make(map[string]bool)
	for _, r := range app.Routes {
		if routeMap[r.Path] {
			return fmt.Errorf("duplicate route definition discovered: %s", r.Path)
		}
		routeMap[r.Path] = true

		// Validate dynamic route parameters
		for _, param := range r.Params {
			if strings.ContainsAny(param, "/\\:*?\"<>|") {
				return fmt.Errorf("malformed dynamic parameter '%s' in route %s", param, r.Path)
			}
		}
	}

	// Formalize and resolve authoritative Mount connections (Resource -> UI Route -> Backend -> Data)
	resourceMounts := make(map[string]*ir.MountIR)

	// 1. Establish Authoritative Resource Mounts from Backend & Data Architecture
	for _, feat := range app.Features {
		if feat == nil || feat.Name == "" {
			continue
		}
		var ent *ir.EntityIR
		for _, e := range app.Entities {
			if strings.EqualFold(e.Name, feat.Entity) || strings.EqualFold(e.Name, feat.Name) {
				ent = e
				break
			}
		}

		pkField := ""
		tableName := ""
		if ent != nil {
			tableName = ent.TableName
			for _, f := range ent.Fields {
				if f.PK {
					pkField = f.Name
					break
				}
				if strings.EqualFold(f.Name, "id") && pkField == "" {
					pkField = f.Name
				}
			}
		}

		if tableName == "" && feat.Repository != "" {
			for _, r := range app.Repositories {
				if r.Name == feat.Repository {
					tableName = r.TargetTable
					break
				}
			}
		}

		resMount := &ir.MountIR{
			ID:         fmt.Sprintf("mount:resource:%s", feat.Name),
			Resource:   feat.Name,
			RoutePath:  feat.Route,
			Entity:     feat.Entity,
			Table:      tableName,
			Repository: feat.Repository,
			Service:    feat.Service,
			Handler:    feat.Handler,
			PrimaryKey: pkField,
			AutoWired:  false,
		}
		resourceMounts[strings.ToLower(feat.Name)] = resMount
		app.Mounts = append(app.Mounts, resMount)
	}

	// Index any entities not explicitly in features
	for _, ent := range app.Entities {
		entKey := strings.ToLower(ent.Name)
		if _, exists := resourceMounts[entKey]; !exists {
			pkField := ""
			for _, f := range ent.Fields {
				if f.PK || strings.EqualFold(f.Name, "id") {
					pkField = f.Name
					break
				}
			}
			repoName := ""
			for _, r := range app.Repositories {
				if strings.EqualFold(r.TargetTable, ent.TableName) || strings.EqualFold(strings.TrimSuffix(r.Name, "Repository"), ent.Name) {
					repoName = r.Name
					break
				}
			}
			svcName := ""
			if repoName != "" {
				for _, s := range app.Services {
					for _, cr := range s.CallsRepos {
						if cr == repoName {
							svcName = s.Name
							break
						}
					}
				}
			}
			handlerName := ""
			if svcName != "" {
				for _, h := range app.Handlers {
					for _, cs := range h.CallsServices {
						if cs == svcName {
							handlerName = h.Name
							break
						}
					}
				}
			}

			resMount := &ir.MountIR{
				ID:         fmt.Sprintf("mount:resource:%s", ent.Name),
				Resource:   ent.Name,
				Entity:     ent.Name,
				Table:      ent.TableName,
				Repository: repoName,
				Service:    svcName,
				Handler:    handlerName,
				PrimaryKey: pkField,
				AutoWired:  false,
			}
			resourceMounts[entKey] = resMount
			app.Mounts = append(app.Mounts, resMount)
		}
	}

	// 2. Bind UI Routes to Mounts
	for _, r := range app.Routes {
		cleanPath := strings.Trim(r.Path, "/")
		parts := strings.Split(cleanPath, "/")
		primarySegment := ""
		if len(parts) > 0 && parts[0] != "" {
			primarySegment = strings.ToLower(parts[0])
		}

		var matchedResource *ir.MountIR
		for resName, rm := range resourceMounts {
			if strings.EqualFold(resName, primarySegment) ||
				strings.EqualFold(resName+"s", primarySegment) ||
				strings.EqualFold(strings.TrimSuffix(primarySegment, "s"), resName) {
				matchedResource = rm
				break
			}
		}

		var routeMount *ir.MountIR
		if matchedResource != nil {
			routeMount = &ir.MountIR{
				ID:         fmt.Sprintf("mount:route:%s", r.Path),
				Resource:   matchedResource.Resource,
				RoutePath:  r.Path,
				Entity:     matchedResource.Entity,
				Table:      matchedResource.Table,
				Repository: matchedResource.Repository,
				Service:    matchedResource.Service,
				Handler:    matchedResource.Handler,
				PrimaryKey: matchedResource.PrimaryKey,
				AutoWired:  false,
			}
		} else {
			tableName := primarySegment
			entityName := ""
			if primarySegment != "" {
				entityName = strings.ToUpper(primarySegment[:1]) + primarySegment[1:]
			}
			routeMount = &ir.MountIR{
				ID:        fmt.Sprintf("mount:route:%s", r.Path),
				Resource:  primarySegment,
				RoutePath: r.Path,
				Table:     tableName,
				Entity:    entityName,
				AutoWired: true,
			}
		}

		if len(r.Params) > 0 {
			routeMount.ParamBinding = r.Params[0]
		}

		app.Mounts = append(app.Mounts, routeMount)
		r.Mount = routeMount
	}

	// 3. Deep Compile-Time Verification of MountIR Relationships
	for _, m := range app.Mounts {
		// 3a. Route parameter ↕ Entity primary key validation
		if m.ParamBinding != "" && m.Entity != "" {
			var boundEntity *ir.EntityIR
			for _, e := range app.Entities {
				if strings.EqualFold(e.Name, m.Entity) {
					boundEntity = e
					break
				}
			}
			if boundEntity != nil && len(boundEntity.Fields) > 0 {
				hasMatchingField := false
				cleanParam := strings.Trim(m.ParamBinding, "[]:")
				for _, f := range boundEntity.Fields {
					if strings.EqualFold(f.Name, cleanParam) || (f.PK && (cleanParam == "id" || strings.HasSuffix(strings.ToLower(cleanParam), "id"))) {
						hasMatchingField = true
						break
					}
				}
				if !hasMatchingField {
					return fmt.Errorf("mount validation error: route parameter '%s' on %s does not match any primary key or field in bound entity '%s'",
						m.ParamBinding, m.RoutePath, m.Entity)
				}
			}
		}

		// 3b. Handler -> Service -> Repository -> Table chain validation
		if m.Handler != "" {
			var hRef *ir.HandlerIR
			for _, h := range app.Handlers {
				if h.Name == m.Handler {
					hRef = h
					break
				}
			}
			if hRef != nil {
				for _, svcName := range hRef.CallsServices {
					foundSvc := false
					for _, s := range app.Services {
						if s.Name == svcName {
							foundSvc = true
							break
						}
					}
					if !foundSvc {
						return fmt.Errorf("mount validation error: handler '%s' bound to mount '%s' calls undeclared service '%s'",
							m.Handler, m.ID, svcName)
					}
				}
			}
		}

		if m.Service != "" {
			var sRef *ir.ServiceIR
			for _, s := range app.Services {
				if s.Name == m.Service {
					sRef = s
					break
				}
			}
			if sRef != nil {
				for _, repoName := range sRef.CallsRepos {
					foundRepo := false
					for _, r := range app.Repositories {
						if r.Name == repoName {
							foundRepo = true
							break
						}
					}
					if !foundRepo {
						return fmt.Errorf("mount validation error: service '%s' bound to mount '%s' calls undeclared repository '%s'",
							m.Service, m.ID, repoName)
					}
				}
			}
		}
	}

	// Validate State fields and transitions if defined
	if app.State != nil {
		fieldNames := make(map[string]bool)
		for _, f := range app.State.Fields {
			if f.Name == "" {
				return fmt.Errorf("state field with empty name discovered in state schema")
			}
			if fieldNames[f.Name] {
				return fmt.Errorf("duplicate state field '%s' discovered in state schema", f.Name)
			}
			fieldNames[f.Name] = true
		}
	}

	return nil
}

func (p *Pipeline) optimizeIR(app *ir.Application) {
	// 1. Deduplicate capability requests
	if len(app.Capabilities) > 1 {
		seen := make(map[ir.CapabilityType]bool)
		var unique []*ir.CapabilityIR
		for _, c := range app.Capabilities {
			if !seen[c.Type] {
				seen[c.Type] = true
				unique = append(unique, c)
			}
		}
		app.Capabilities = unique
	}

	// 2. Normalize and order mounts
	mountIndex := make(map[string]*ir.MountIR)
	for _, m := range app.Mounts {
		mountIndex[m.RoutePath] = m
	}
	for _, r := range app.Routes {
		if m, ok := mountIndex[r.Path]; ok {
			r.Mount = m
		}
	}
}
