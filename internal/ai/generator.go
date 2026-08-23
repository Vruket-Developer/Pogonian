package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GeneratedArtifacts holds the files created by the AI Code Generator.
type GeneratedArtifacts struct {
	ModuleName string
	Files      []string
}

// ApplyPlan generates and writes all multi-layer code files for an AI plan into the project.
func ApplyPlan(projectDir string, plan *AIPlan) (*GeneratedArtifacts, error) {
	mod := strings.ToLower(plan.FeatureName)
	pascal := strings.ToUpper(mod[:1]) + mod[1:]

	artifacts := &GeneratedArtifacts{
		ModuleName: mod,
		Files:      make([]string, 0),
	}

	serverModDir := filepath.Join(projectDir, "server", mod)
	appModDir := filepath.Join(projectDir, "app", mod)

	_ = os.MkdirAll(serverModDir, 0755)
	_ = os.MkdirAll(appModDir, 0755)

	// 1. Entity: server/<mod>/entity.go
	entityCode := fmt.Sprintf(`package %s

import "time"

// %s represents the domain entity for %s in Pogonian.
type %s struct {
	ID        int64     `+"`json:\"id\"`"+`
	Title     string    `+"`json:\"title\"`"+`
	Content   string    `+"`json:\"content\"`"+`
	CreatedAt time.Time `+"`json:\"created_at\"`"+`
}
`, mod, pascal, mod, pascal)
	entityPath := filepath.Join(serverModDir, "entity.go")
	if err := os.WriteFile(entityPath, []byte(entityCode), 0644); err != nil {
		return nil, err
	}
	artifacts.Files = append(artifacts.Files, entityPath)

	// 2. Repository: server/<mod>/repository.go
	repoCode := fmt.Sprintf(`package %s

import (
	"database/sql"
	"fmt"
	"time"
)

// Repository manages database operations for %s.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new %s Repository.
func NewRepository(db *sql.DB) (*Repository, error) {
	repo := &Repository{db: db}
	if err := repo.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate %s table: %%w", err)
	}
	return repo, nil
}

func (r *Repository) migrate() error {
	query := `+"`"+`
	CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		content TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`+"`"+`
	_, err := r.db.Exec(query)
	return err
}

// FindAll returns all %s records.
func (r *Repository) FindAll() ([]*%s, error) {
	rows, err := r.db.Query("SELECT id, title, content, created_at FROM %s ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*%s
	for rows.Next() {
		item := &%s{}
		var createdAtStr string
		if err := rows.Scan(&item.ID, &item.Title, &item.Content, &createdAtStr); err != nil {
			return nil, err
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		list = append(list, item)
	}
	return list, nil
}

// Create inserts a new %s record.
func (r *Repository) Create(title, content string) (*%s, error) {
	res, err := r.db.Exec("INSERT INTO %s (title, content, created_at) VALUES (?, ?, ?)",
		title, content, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &%s{
		ID:        id,
		Title:     title,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	}, nil
}
`, mod, pascal, pascal, mod, mod, mod, pascal, mod, pascal, pascal, pascal, pascal, mod, pascal)
	repoPath := filepath.Join(serverModDir, "repository.go")
	if err := os.WriteFile(repoPath, []byte(repoCode), 0644); err != nil {
		return nil, err
	}
	artifacts.Files = append(artifacts.Files, repoPath)

	// 3. Service: server/<mod>/service.go
	serviceCode := fmt.Sprintf(`package %s

import (
	"errors"
	"strings"
)

// Service encapsulates domain business logic for %s.
type Service struct {
	repo *Repository
}

// NewService creates a new %s Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// List retrieves all %s records.
func (s *Service) List() ([]*%s, error) {
	return s.repo.FindAll()
}

// Create validates and creates a %s record.
func (s *Service) Create(title, content string) (*%s, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.New("title cannot be empty")
	}
	return s.repo.Create(title, content)
}
`, mod, pascal, pascal, mod, pascal, pascal, pascal)
	servicePath := filepath.Join(serverModDir, "service.go")
	if err := os.WriteFile(servicePath, []byte(serviceCode), 0644); err != nil {
		return nil, err
	}
	artifacts.Files = append(artifacts.Files, servicePath)

	// 4. Handler: server/<mod>/handler.go
	handlerCode := fmt.Sprintf(`package %s

import (
	"net/http"
)

// Handler handles HTTP requests and form actions for %s.
type Handler struct {
	service *Service
}

// NewHandler creates a new %s HTTP Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// HandleGet returns %s data for page rendering.
func (h *Handler) HandleGet(r *http.Request, params map[string]string) (interface{}, error) {
	items, err := h.service.List()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"Items": items,
		"Total": len(items),
	}, nil
}

// HandlePostAction handles form submissions.
func (h *Handler) HandlePostAction(w http.ResponseWriter, r *http.Request, params map[string]string) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	title := r.FormValue("title")
	content := r.FormValue("content")

	if _, err := h.service.Create(title, content); err != nil {
		return err
	}

	http.Redirect(w, r, "/%s", http.StatusSeeOther)
	return nil
}
`, mod, pascal, pascal, mod, mod)
	handlerPath := filepath.Join(serverModDir, "handler.go")
	if err := os.WriteFile(handlerPath, []byte(handlerCode), 0644); err != nil {
		return nil, err
	}
	artifacts.Files = append(artifacts.Files, handlerPath)

	// 5. Frontend HTML: app/<mod>/page.html
	htmlCode := fmt.Sprintf(`<section class="content-section">
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 2rem;">
        <div>
            <h1>%s</h1>
            <p style="color: var(--text-secondary);">Managed by Pogonian Architecture: <code>Page &rarr; Form Action &rarr; Handler &rarr; Service &rarr; Repository &rarr; SQLite</code></p>
        </div>
    </div>

    <div style="display: grid; grid-template-columns: 1fr 1.5fr; gap: 2rem;">
        <div class="card">
            <h3>Add New %s</h3>
            <form action="/%s" method="post" style="display: flex; flex-direction: column; gap: 1rem; margin-top: 1rem;">
                <input type="hidden" name="_csrf" value="{{ .CSRFToken }}">
                <div>
                    <label style="display: block; margin-bottom: 0.35rem; font-weight: 500;">Title</label>
                    <input type="text" name="title" placeholder="Enter title..." required
                           style="width: 100%%; padding: 0.65rem 0.85rem; border-radius: 6px; border: 1px solid var(--border-color); background: var(--bg-surface-elevated); color: var(--text-primary);">
                </div>
                <div>
                    <label style="display: block; margin-bottom: 0.35rem; font-weight: 500;">Content</label>
                    <textarea name="content" rows="3" placeholder="Enter details..."
                              style="width: 100%%; padding: 0.65rem 0.85rem; border-radius: 6px; border: 1px solid var(--border-color); background: var(--bg-surface-elevated); color: var(--text-primary);"></textarea>
                </div>
                <button type="submit" class="btn btn-primary">+ Submit %s</button>
            </form>
        </div>

        <div class="card">
            <h3>%s Items</h3>
            <div style="margin-top: 1rem;">
                <table style="width: 100%%; border-collapse: collapse;">
                    <thead>
                        <tr style="border-bottom: 1px solid var(--border-color); color: var(--text-muted); font-size: 0.85rem;">
                            <th style="padding: 0.75rem 0.5rem;">ID</th>
                            <th style="padding: 0.75rem 0.5rem;">TITLE</th>
                            <th style="padding: 0.75rem 0.5rem;">CONTENT</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{ if .Data.Items }}
                            {{ range .Data.Items }}
                            <tr style="border-bottom: 1px solid var(--border-color);">
                                <td style="padding: 0.75rem 0.5rem; color: var(--text-muted);">#{{ .ID }}</td>
                                <td style="padding: 0.75rem 0.5rem; font-weight: 600;">{{ .Title }}</td>
                                <td style="padding: 0.75rem 0.5rem; color: var(--text-secondary);">{{ .Content }}</td>
                            </tr>
                            {{ end }}
                        {{ else }}
                            <tr>
                                <td colspan="3" style="padding: 2rem; text-align: center; color: var(--text-muted);">
                                    No %s entries found. Use the form on the left to add one!
                                </td>
                            </tr>
                        {{ end }}
                    </tbody>
                </table>
            </div>
        </div>
    </div>
</section>
`, pascal, pascal, mod, pascal, pascal, mod)
	htmlPath := filepath.Join(appModDir, "page.html")
	if err := os.WriteFile(htmlPath, []byte(htmlCode), 0644); err != nil {
		return nil, err
	}
	artifacts.Files = append(artifacts.Files, htmlPath)

	// 6. Update .pogonian/architecture/app.ai
	aiPath := filepath.Join(projectDir, ".pogonian", "architecture", "app.ai")
	if data, err := os.ReadFile(aiPath); err == nil {
		content := string(data)
		if !strings.Contains(content, mod) {
			newContent := content + fmt.Sprintf("  - %s\n", mod)
			_ = os.WriteFile(aiPath, []byte(newContent), 0644)
		}
	}

	return artifacts, nil
}
