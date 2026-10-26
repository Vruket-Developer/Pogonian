package modules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pogonian/internal/ai"
	"pogonian/pkg/dataruntime"
)

// ModuleCategory specifies if a module is UI-only or full-stack.
type ModuleCategory string

const (
	CategoryUIComponent ModuleCategory = "UI Component"
	CategoryFullStack   ModuleCategory = "Full-Stack Feature"
)

// Module represents a downloadable/scaffoldable component or feature module in the Pogonian Marketplace.
type Module struct {
	Name        string            `json:"name"`
	Category    ModuleCategory    `json:"category"`
	Description string            `json:"description"`
	Files       map[string]string `json:"files"`
	CSS         string            `json:"css,omitempty"`
	Migration   string            `json:"migration,omitempty"`
	UsageHint   string            `json:"usage_hint"`
}

// InstallResult details the installed files and instructions.
type InstallResult struct {
	ModuleName   string
	Category     ModuleCategory
	FilesCreated []string
	UsageHint    string
}

// RegistryProvider resolves modules from a source (builtin catalog, local directory, remote git).
type RegistryProvider interface {
	Name() string
	Get(name string) (*Module, error)
	List() []*Module
}

// BuiltinProvider serves modules from the embedded catalog.
type BuiltinProvider struct{}

func (b *BuiltinProvider) Name() string { return "builtin" }

func (b *BuiltinProvider) Get(name string) (*Module, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	if m, ok := Catalog[key]; ok {
		return m, nil
	}
	return nil, fmt.Errorf("module '%s' not found in builtin catalog", name)
}

func (b *BuiltinProvider) List() []*Module {
	var list []*Module
	for _, m := range Catalog {
		list = append(list, m)
	}
	return list
}

// GlobalRegistry manages active registry providers.
var GlobalRegistry RegistryProvider = &BuiltinProvider{}

// Catalog holds all available built-in marketplace components and modules.
var Catalog = map[string]*Module{
	"header": {
		Name:        "header",
		Category:    CategoryUIComponent,
		Description: "Responsive navigation header with branding, active route links, and mobile drawer toggle",
		UsageHint:   `Add {{ template "components/header.html" . }} or {{ template "header" . }} at the top of app/layout.html.`,
		Files: map[string]string{
			"app/components/header.html": `{{ define "header" }}
<header class="pogn-component-header">
    <div class="container nav-container">
        <a href="/" class="brand-logo">
            <span class="logo-badge">P</span>
            <strong>Pogonian</strong>
        </a>
        <nav class="nav-links">
            <a href="/" class="nav-link">Home</a>
            <a href="/products" class="nav-link">Products</a>
            <a href="/architecture" class="nav-link">Architecture</a>
            <a href="/feedback" class="nav-link">Feedback</a>
            <a href="/login" class="btn btn-outline" style="margin-left: 1rem;">Sign In</a>
            <a href="/__pogn/devtools" class="nav-link devtools-badge">⚡ DevTools</a>
        </nav>
    </div>
</header>
{{ end }}`,
		},
	},

	"footer": {
		Name:        "footer",
		Category:    CategoryUIComponent,
		Description: "Responsive site footer with multi-column links, copyright notice, and branding",
		UsageHint:   `Add {{ template "components/footer.html" . }} or {{ template "footer" . }} at the bottom of app/layout.html.`,
		Files: map[string]string{
			"app/components/footer.html": `{{ define "footer" }}
<footer class="pogn-component-footer">
    <div class="container footer-container">
        <div class="footer-brand">
            <div class="brand-logo">
                <span class="logo-badge">P</span>
                <strong>Pogonian</strong>
            </div>
            <p class="footer-tagline">Architecture-Aware Full-Stack Go & HTML-First Web Platform.</p>
        </div>
        <div class="footer-links">
            <div class="footer-col">
                <h4>Platform</h4>
                <a href="/">Home</a>
                <a href="/products">Products</a>
                <a href="/architecture">Architecture</a>
            </div>
            <div class="footer-col">
                <h4>Resources</h4>
                <a href="/feedback">Feedback</a>
                <a href="/login">Sign In</a>
                <a href="/__pogn/devtools">⚡ DevTools</a>
            </div>
        </div>
    </div>
    <div class="container footer-bottom">
        <p>&copy; 2026 Pogonian. All rights reserved.</p>
    </div>
</footer>
{{ end }}`,
		},
		CSS: `/* Pogonian Marketplace: Footer */
.pogn-component-footer {
    border-top: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
    background: var(--bg-surface, #12161f);
    padding: 3rem 0 2rem;
    margin-top: 4rem;
}
.footer-container {
    display: flex;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 2rem;
    margin-bottom: 2rem;
}
.footer-brand .footer-tagline {
    color: var(--text-secondary, #9aa5b5);
    font-size: 0.9rem;
    margin-top: 0.5rem;
    max-width: 280px;
    line-height: 1.5;
}
.footer-links {
    display: flex;
    gap: 3rem;
}
.footer-col h4 {
    font-size: 0.95rem;
    font-weight: 600;
    margin-bottom: 0.75rem;
    color: var(--text-primary, #f0f3f6);
}
.footer-col a {
    display: block;
    color: var(--text-secondary, #9aa5b5);
    font-size: 0.875rem;
    margin-bottom: 0.5rem;
    text-decoration: none;
    transition: var(--transition, all 0.2s ease);
}
.footer-col a:hover {
    color: var(--brand-primary, #38bdf8);
}
.footer-bottom {
    border-top: 1px solid var(--border-color, rgba(255, 255, 255, 0.05));
    padding-top: 1.5rem;
    font-size: 0.85rem;
    color: var(--text-muted, #626d7d);
    text-align: center;
}
`,
	},

	"sidebar": {
		Name:        "sidebar",
		Category:    CategoryUIComponent,
		Description: "Collapsible dashboard sidebar with icon navigation items, badges, and user profile drawer",
		UsageHint:   `Add {{ template "components/sidebar.html" . }} in dashboard views or side-by-side layouts.`,
		Files: map[string]string{
			"app/components/sidebar.html": `{{ define "sidebar" }}
<aside class="pogn-component-sidebar">
    <div class="sidebar-header">
        <span class="sidebar-title">Dashboard</span>
    </div>
    <ul class="sidebar-menu">
        <li class="sidebar-item"><a href="/" class="sidebar-link active">📊 Overview</a></li>
        <li class="sidebar-item"><a href="/products" class="sidebar-link">📦 Inventory <span class="badge">Live</span></a></li>
        <li class="sidebar-item"><a href="/architecture" class="sidebar-link">🏛️ Architecture Graph</a></li>
        <li class="sidebar-item"><a href="/feedback" class="sidebar-link">💬 User Feedback</a></li>
        <li class="sidebar-item"><a href="/__pogn/devtools" class="sidebar-link">⚡ DevTools HUD</a></li>
    </ul>
</aside>
{{ end }}`,
		},
	},

	"card": {
		Name:        "card",
		Category:    CategoryUIComponent,
		Description: "Interactive content card component with header, body slot, and call-to-action buttons",
		UsageHint:   `Add {{ template "components/card.html" . }} anywhere to display cards.`,
		Files: map[string]string{
			"app/components/card.html": `{{ define "card" }}
<div class="pogn-component-card card">
    <div class="card-header">
        <h3>{{ if .CardTitle }}{{ .CardTitle }}{{ else }}Featured Module{{ end }}</h3>
    </div>
    <div class="card-body">
        <p>{{ if .CardBody }}{{ .CardBody }}{{ else }}Clean, modular HTML-first component rendered natively by Go.{{ end }}</p>
    </div>
    <div class="card-actions">
        <a href="{{ if .CardLink }}{{ .CardLink }}{{ else }}#{{ end }}" class="btn btn-primary">Learn More &rarr;</a>
    </div>
</div>
{{ end }}`,
		},
	},

	"login": {
		Name:        "login",
		Category:    CategoryFullStack,
		Description: "Full-stack authentication with Login UI, Sign Up, Argon2id password hashing, and session management",
		UsageHint:   `Access at /login and /register. Database tables created in data.db.`,
		Files: map[string]string{
			"app/login/page.html": `<section class="content-section" style="max-width: 480px; margin: 3rem auto;">
    <div class="card" style="padding: 2.5rem;">
        <h1 style="font-size: 1.8rem; margin-bottom: 0.5rem; text-align: center;">Sign In</h1>
        <p style="color: var(--text-secondary); text-align: center; margin-bottom: 2rem;">Access your Pogonian application</p>
        
        <form action="/login" method="post">
            <input type="hidden" name="_csrf" value="{{ .CSRFToken }}">
            
            <div class="form-group" style="margin-bottom: 1.25rem;">
                <label for="email" style="display: block; margin-bottom: 0.5rem; font-weight: 500;">Email Address</label>
                <input type="email" id="email" name="email" required placeholder="you@example.com" class="input-field" style="width: 100%; padding: 0.75rem; border-radius: 8px; background: var(--bg-main, #09090b); border: 1px solid var(--border-color, rgba(255,255,255,0.15)); color: #fff;">
            </div>
            
            <div class="form-group" style="margin-bottom: 1.5rem;">
                <div style="display: flex; justify-content: space-between; margin-bottom: 0.5rem;">
                    <label for="password" style="font-weight: 500;">Password</label>
                    <a href="/forgot-password" style="font-size: 0.85rem; color: var(--primary, #38bdf8); text-decoration: none;">Forgot password?</a>
                </div>
                <input type="password" id="password" name="password" required placeholder="••••••••" class="input-field" style="width: 100%; padding: 0.75rem; border-radius: 8px; background: var(--bg-main, #09090b); border: 1px solid var(--border-color, rgba(255,255,255,0.15)); color: #fff;">
            </div>
            
            <button type="submit" class="btn btn-primary" style="width: 100%; padding: 0.85rem; font-weight: 600;">Sign In</button>
        </form>
        
        <div style="text-align: center; margin-top: 1.5rem; font-size: 0.9rem; color: var(--text-secondary);">
            Don't have an account? <a href="/register" style="color: var(--primary, #38bdf8); font-weight: 500;">Create one</a>
        </div>
    </div>
</section>`,
			"app/register/page.html": `<section class="content-section" style="max-width: 480px; margin: 3rem auto;">
    <div class="card" style="padding: 2.5rem;">
        <h1 style="font-size: 1.8rem; margin-bottom: 0.5rem; text-align: center;">Create Account</h1>
        <p style="color: var(--text-secondary); text-align: center; margin-bottom: 2rem;">Get started with Pogonian in seconds</p>
        
        <form action="/register" method="post">
            <input type="hidden" name="_csrf" value="{{ .CSRFToken }}">
            
            <div class="form-group" style="margin-bottom: 1.25rem;">
                <label for="name" style="display: block; margin-bottom: 0.5rem; font-weight: 500;">Full Name</label>
                <input type="text" id="name" name="name" required placeholder="Alex Mercer" class="input-field" style="width: 100%; padding: 0.75rem; border-radius: 8px; background: var(--bg-main, #09090b); border: 1px solid var(--border-color, rgba(255,255,255,0.15)); color: #fff;">
            </div>

            <div class="form-group" style="margin-bottom: 1.25rem;">
                <label for="email" style="display: block; margin-bottom: 0.5rem; font-weight: 500;">Email Address</label>
                <input type="email" id="email" name="email" required placeholder="you@example.com" class="input-field" style="width: 100%; padding: 0.75rem; border-radius: 8px; background: var(--bg-main, #09090b); border: 1px solid var(--border-color, rgba(255,255,255,0.15)); color: #fff;">
            </div>
            
            <div class="form-group" style="margin-bottom: 1.5rem;">
                <label for="password" style="display: block; margin-bottom: 0.5rem; font-weight: 500;">Password (min 8 chars)</label>
                <input type="password" id="password" name="password" required minlength="8" placeholder="••••••••" class="input-field" style="width: 100%; padding: 0.75rem; border-radius: 8px; background: var(--bg-main, #09090b); border: 1px solid var(--border-color, rgba(255,255,255,0.15)); color: #fff;">
            </div>
            
            <button type="submit" class="btn btn-primary" style="width: 100%; padding: 0.85rem; font-weight: 600;">Create Account</button>
        </form>
        
        <div style="text-align: center; margin-top: 1.5rem; font-size: 0.9rem; color: var(--text-secondary);">
            Already have an account? <a href="/login" style="color: var(--primary, #38bdf8); font-weight: 500;">Sign in</a>
        </div>
    </div>
</section>`,
		},
		Migration: `CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    name TEXT,
    role TEXT DEFAULT 'user',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    user_email TEXT NOT NULL,
    user_name TEXT,
    user_role TEXT,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL
);`,
	},
}

// ListModules returns all marketplace modules sorted by category.
func ListModules() []*Module {
	var list []*Module
	for _, m := range Catalog {
		list = append(list, m)
	}
	return list
}

// InstallModule installs a marketplace module into the target project directory.
func InstallModule(projectDir, moduleName string) (*InstallResult, error) {
	modKey := strings.ToLower(strings.TrimSpace(moduleName))
	mod, exists := Catalog[modKey]
	if !exists {
		return nil, fmt.Errorf("module '%s' not found in Pogonian Marketplace. Run 'pogn modules' to see available modules", moduleName)
	}

	result := &InstallResult{
		ModuleName:   mod.Name,
		Category:     mod.Category,
		FilesCreated: make([]string, 0),
		UsageHint:    mod.UsageHint,
	}

	// 1. Create Files
	for relPath, content := range mod.Files {
		destPath := filepath.Join(projectDir, relPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(destPath, []byte(content), 0644); err != nil {
			return nil, err
		}
		result.FilesCreated = append(result.FilesCreated, destPath)
	}

	// 2. Append CSS if provided
	if mod.CSS != "" {
		cssPath := filepath.Join(projectDir, "globals.css")
		if _, err := os.Stat(cssPath); os.IsNotExist(err) {
			if _, appErr := os.Stat(filepath.Join(projectDir, "app", "globals.css")); appErr == nil {
				cssPath = filepath.Join(projectDir, "app", "globals.css")
			}
		}
		if _, err := os.Stat(cssPath); err == nil {
			existing, _ := os.ReadFile(cssPath)
			if !strings.Contains(string(existing), "Pogonian Marketplace: "+strings.Title(mod.Name)) {
				newCSS := string(existing) + "\n" + mod.CSS
				_ = os.WriteFile(cssPath, []byte(newCSS), 0644)
			}
		}
	}

	// 3. Create Migration if needed
	if mod.Migration != "" {
		migDir := filepath.Join(projectDir, "migrations")
		_ = os.MkdirAll(migDir, 0755)
		upFile, downFile, err := dataruntime.GenerateMigration(migDir, "create_"+mod.Name)
		if err == nil {
			_ = os.WriteFile(upFile, []byte(mod.Migration), 0644)
			_ = os.WriteFile(downFile, []byte("DROP TABLE IF EXISTS "+mod.Name+";"), 0644)
			result.FilesCreated = append(result.FilesCreated, upFile, downFile)
		}
	}

	// 4. Register in .pogonian/architecture/app.ai
	spec, err := ai.ParseAISpec(projectDir)
	if err == nil && spec != nil {
		spec.AddModule(mod.Name)
		_ = spec.Save(projectDir)
	}

	return result, nil
}
