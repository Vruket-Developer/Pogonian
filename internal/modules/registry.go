package modules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pogonian/internal/ai"
	"pogonian/internal/database"
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

// Catalog holds all available built-in marketplace components and modules.
var Catalog = map[string]*Module{
	"footer": {
		Name:        "footer",
		Category:    CategoryUIComponent,
		Description: "Modern responsive HTML5 footer with brand logo, nav links, social icons, and copyright",
		UsageHint:   `Add {{ template "components/footer.html" . }} or {{ template "footer" . }} into app/layout.html or any page.`,
		Files: map[string]string{
			"app/components/footer.html": `{{ define "footer" }}
<footer class="pogn-component-footer">
    <div class="container footer-grid">
        <div class="footer-brand-col">
            <div class="brand-logo">
                <span class="logo-badge">P</span>
                <strong>Pogonian App</strong>
            </div>
            <p class="footer-tagline">High-performance, HTML-first web application powered by Go & Radix routing.</p>
            <div class="footer-socials">
                <a href="https://github.com" target="_blank" aria-label="GitHub" class="social-link">GitHub</a>
                <a href="https://twitter.com" target="_blank" aria-label="Twitter" class="social-link">Twitter</a>
                <a href="https://discord.com" target="_blank" aria-label="Discord" class="social-link">Discord</a>
            </div>
        </div>
        
        <div class="footer-links-col">
            <h4>Product</h4>
            <ul>
                <li><a href="/products">Features & Catalog</a></li>
                <li><a href="/pricing">Pricing Plans</a></li>
                <li><a href="/architecture">Architecture Flow</a></li>
                <li><a href="/__pogn/devtools">Live DevTools</a></li>
            </ul>
        </div>
        
        <div class="footer-links-col">
            <h4>Resources</h4>
            <ul>
                <li><a href="/docs">Documentation</a></li>
                <li><a href="/about">About Us</a></li>
                <li><a href="/feedback">Feedback & Support</a></li>
                <li><a href="/privacy">Privacy Policy</a></li>
            </ul>
        </div>
        
        <div class="footer-newsletter-col">
            <h4>Stay Connected</h4>
            <p>Subscribe for product updates and engineering deep dives.</p>
            <form action="/newsletter" method="post" class="footer-newsletter-form">
                <input type="hidden" name="_csrf" value="{{ .CSRFToken }}">
                <div class="newsletter-input-group">
                    <input type="email" name="email" placeholder="you@example.com" required class="input-field">
                    <button type="submit" class="btn btn-primary">Subscribe</button>
                </div>
            </form>
        </div>
    </div>
    
    <div class="footer-bottom">
        <div class="container footer-bottom-content">
            <p>&copy; 2026 Pogonian Framework. Built with HTML-First architecture & Go backend.</p>
            <div class="footer-bottom-links">
                <a href="/terms">Terms</a>
                <a href="/privacy">Privacy</a>
                <a href="/security">Security</a>
            </div>
        </div>
    </div>
</footer>
{{ end }}`,
		},
		CSS: `
/* Pogonian Marketplace: Footer Component Styles */
.pogn-component-footer {
    background: var(--bg-card, #18181b);
    border-top: 1px solid var(--border-color, rgba(255,255,255,0.08));
    color: var(--text-secondary, #a1a1aa);
    padding-top: 3.5rem;
    margin-top: 4rem;
}
.footer-grid {
    display: grid;
    grid-template-columns: 2fr 1fr 1fr 1.5fr;
    gap: 2.5rem;
    padding-bottom: 3rem;
}
@media (max-width: 900px) {
    .footer-grid {
        grid-template-columns: 1fr 1fr;
    }
}
@media (max-width: 600px) {
    .footer-grid {
        grid-template-columns: 1fr;
    }
}
.footer-brand-col .footer-tagline {
    margin: 1rem 0 1.5rem;
    line-height: 1.6;
    font-size: 0.95rem;
}
.footer-socials {
    display: flex;
    gap: 1rem;
}
.footer-socials .social-link {
    color: var(--text-primary, #f4f4f5);
    text-decoration: none;
    font-weight: 500;
    font-size: 0.9rem;
    padding: 0.25rem 0.5rem;
    background: rgba(255,255,255,0.05);
    border-radius: 6px;
    transition: all 0.2s ease;
}
.footer-socials .social-link:hover {
    background: var(--primary, #38bdf8);
    color: #000;
}
.footer-links-col h4, .footer-newsletter-col h4 {
    color: var(--text-primary, #f4f4f5);
    font-size: 1rem;
    font-weight: 600;
    margin-bottom: 1.25rem;
}
.footer-links-col ul {
    list-style: none;
    padding: 0;
    margin: 0;
}
.footer-links-col ul li {
    margin-bottom: 0.75rem;
}
.footer-links-col ul li a {
    color: var(--text-secondary, #a1a1aa);
    text-decoration: none;
    font-size: 0.9rem;
    transition: color 0.2s ease;
}
.footer-links-col ul li a:hover {
    color: var(--primary, #38bdf8);
}
.newsletter-input-group {
    display: flex;
    gap: 0.5rem;
    margin-top: 1rem;
}
.newsletter-input-group .input-field {
    flex: 1;
    padding: 0.6rem 0.8rem;
    background: var(--bg-main, #09090b);
    border: 1px solid var(--border-color, rgba(255,255,255,0.12));
    border-radius: 8px;
    color: #fff;
}
.footer-bottom {
    border-top: 1px solid var(--border-color, rgba(255,255,255,0.08));
    padding: 1.5rem 0;
    font-size: 0.85rem;
}
.footer-bottom-content {
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 1rem;
}
.footer-bottom-links {
    display: flex;
    gap: 1.5rem;
}
.footer-bottom-links a {
    color: var(--text-secondary, #a1a1aa);
    text-decoration: none;
}
.footer-bottom-links a:hover {
    color: var(--primary, #38bdf8);
}
`,
	},

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
		upFile, downFile, err := database.GenerateMigration(migDir, "create_"+mod.Name)
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
