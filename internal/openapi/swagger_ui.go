package openapi

import (
	"fmt"
	"net/http"
	"path/filepath"

	"pogonian/internal/config"
)

// SwaggerUIHTML returns a sleek, dark-mode Swagger UI client with responsive controls and interactive testing.
func SwaggerUIHTML(appName, specURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s | Interactive API & Route Docs</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css" />
    <style>
        :root {
            --pogn-bg: #0b0f19;
            --pogn-card: #111827;
            --pogn-border: #1f2937;
            --pogn-primary: #38bdf8;
            --pogn-primary-hover: #0ea5e9;
            --pogn-accent: #818cf8;
            --pogn-text: #f3f4f6;
            --pogn-text-muted: #9ca3af;
        }

        body {
            margin: 0;
            padding: 0;
            background: var(--pogn-bg);
            color: var(--pogn-text);
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
        }

        /* Top Navigation Header */
        .pogn-header {
            background: rgba(17, 24, 39, 0.95);
            backdrop-filter: blur(12px);
            border-bottom: 1px solid var(--pogn-border);
            padding: 0.85rem 2rem;
            display: flex;
            align-items: center;
            justify-content: space-between;
            position: sticky;
            top: 0;
            z-index: 100;
        }

        .pogn-header-brand {
            display: flex;
            align-items: center;
            gap: 12px;
            text-decoration: none;
            color: var(--pogn-text);
        }

        .pogn-badge {
            background: linear-gradient(135deg, var(--pogn-primary), var(--pogn-accent));
            color: #fff;
            font-weight: 800;
            font-size: 0.75rem;
            padding: 4px 8px;
            border-radius: 6px;
            letter-spacing: 0.05em;
        }

        .pogn-header-title {
            font-size: 1.1rem;
            font-weight: 700;
            letter-spacing: -0.01em;
        }

        .pogn-nav-links {
            display: flex;
            align-items: center;
            gap: 1rem;
        }

        .pogn-nav-link {
            color: var(--pogn-text-muted);
            text-decoration: none;
            font-size: 0.875rem;
            font-weight: 500;
            padding: 6px 12px;
            border-radius: 6px;
            transition: all 0.15s ease;
        }

        .pogn-nav-link:hover {
            color: var(--pogn-primary);
            background: rgba(56, 189, 248, 0.08);
        }

        .pogn-btn-spec {
            background: rgba(56, 189, 248, 0.15);
            border: 1px solid rgba(56, 189, 248, 0.4);
            color: var(--pogn-primary);
            font-weight: 600;
        }

        /* Swagger UI Overrides for Dark Modern Aesthetic */
        #swagger-ui {
            max-width: 1300px;
            margin: 0 auto;
            padding: 1.5rem 1rem 4rem 1rem;
        }

        .swagger-ui .topbar { display: none; }
        
        .swagger-ui {
            color: var(--pogn-text);
        }

        .swagger-ui .info {
            margin: 20px 0;
        }

        .swagger-ui .info .title {
            color: #fff;
            font-weight: 800;
            letter-spacing: -0.02em;
        }

        .swagger-ui .info p, .swagger-ui .info li {
            color: var(--pogn-text-muted);
        }

        .swagger-ui .scheme-container {
            background: var(--pogn-card);
            box-shadow: none;
            border: 1px solid var(--pogn-border);
            border-radius: 8px;
            padding: 15px;
            margin-bottom: 20px;
        }

        .swagger-ui .opblock-tag {
            color: var(--pogn-primary) !important;
            border-bottom: 1px solid var(--pogn-border);
            font-size: 1.25rem;
            font-weight: 700;
            padding: 10px 0;
            margin: 20px 0 10px 0;
        }

        .swagger-ui .opblock-tag small {
            color: var(--pogn-text-muted);
            font-weight: normal;
        }

        .swagger-ui .opblock {
            border-radius: 8px;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
            margin: 0 0 14px;
            border: 1px solid var(--pogn-border);
            background: var(--pogn-card);
        }

        .swagger-ui .opblock .opblock-summary {
            padding: 8px 12px;
        }

        .swagger-ui .opblock .opblock-summary-path {
            color: #fff;
            font-weight: 600;
            font-size: 0.95rem;
        }

        .swagger-ui .opblock .opblock-summary-description {
            color: var(--pogn-text-muted);
            font-size: 0.85rem;
        }

        .swagger-ui .opblock.opblock-get {
            border-color: rgba(56, 189, 248, 0.3);
            background: rgba(56, 189, 248, 0.03);
        }
        .swagger-ui .opblock.opblock-get .opblock-summary-method {
            background: #0284c7;
        }

        .swagger-ui .opblock.opblock-post {
            border-color: rgba(52, 211, 153, 0.3);
            background: rgba(52, 211, 153, 0.03);
        }
        .swagger-ui .opblock.opblock-post .opblock-summary-method {
            background: #059669;
        }

        .swagger-ui .opblock.opblock-put {
            border-color: rgba(251, 191, 36, 0.3);
            background: rgba(251, 191, 36, 0.03);
        }
        .swagger-ui .opblock.opblock-put .opblock-summary-method {
            background: #d97706;
        }

        .swagger-ui .opblock.opblock-delete {
            border-color: rgba(248, 113, 113, 0.3);
            background: rgba(248, 113, 113, 0.03);
        }
        .swagger-ui .opblock.opblock-delete .opblock-summary-method {
            background: #dc2626;
        }

        .swagger-ui .opblock-body {
            background: #0d131f;
            border-top: 1px solid var(--pogn-border);
        }

        .swagger-ui table thead tr th, .swagger-ui table thead tr td {
            color: #fff;
            border-bottom: 1px solid var(--pogn-border);
        }

        .swagger-ui .parameter__name {
            color: #fff;
            font-weight: 600;
        }

        .swagger-ui .parameter__type {
            color: var(--pogn-primary);
        }

        .swagger-ui section.models {
            border: 1px solid var(--pogn-border);
            border-radius: 8px;
            background: var(--pogn-card);
        }

        .swagger-ui section.models h4 {
            color: var(--pogn-primary);
        }

        .swagger-ui .model-box {
            background: #0b0f19;
            border-radius: 6px;
        }

        .swagger-ui .btn {
            border-radius: 6px;
            font-weight: 600;
        }

        .swagger-ui .btn.execute {
            background-color: var(--pogn-primary);
            border-color: var(--pogn-primary);
            color: #0b0f19;
        }
    </style>
</head>
<body>
    <header class="pogn-header">
        <a href="/" class="pogn-header-brand">
            <span class="pogn-badge">POGONIAN</span>
            <span class="pogn-header-title">%s API & Swagger</span>
        </a>
        <nav class="pogn-nav-links">
            <a href="/" class="pogn-nav-link">&larr; Back to App</a>
            <a href="/__pogn/devtools" class="pogn-nav-link">⚡ DevTools</a>
            <a href="%s" target="_blank" class="pogn-nav-link pogn-btn-spec">OpenAPI JSON ↗</a>
        </nav>
    </header>

    <div id="swagger-ui"></div>

    <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            window.ui = SwaggerUIBundle({
                url: "%s",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "BaseLayout",
                persistAuthorization: true,
                displayRequestDuration: true,
                filter: true,
                tryItOutEnabled: true,
                syntaxHighlight: {
                    theme: "nord"
                }
            });
        };
    </script>
</body>
</html>`, appName, appName, specURL, specURL)
}

// ServeSwaggerUI returns an http.HandlerFunc that renders the interactive Swagger UI page.
func ServeSwaggerUI(projectDir string, specURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		absDir, _ := filepath.Abs(projectDir)
		cfg, _ := config.Load(absDir)
		appName := "Pogonian App"
		if cfg != nil && cfg.Name != "" {
			appName = cfg.Name
		}

		if specURL == "" {
			specURL = "/swagger.json"
		}

		html := SwaggerUIHTML(appName, specURL)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	}
}
