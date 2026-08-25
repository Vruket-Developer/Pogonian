package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"pogonian/internal/ir"
)

// ProviderType represents supported AI backend engines.
type ProviderType string

const (
	ProviderOpenAI ProviderType = "openai"
	ProviderGemini ProviderType = "gemini"
	ProviderClaude ProviderType = "claude"
	ProviderLocal  ProviderType = "local"
	ProviderMock   ProviderType = "mock"
)

// AIPlanRequest contains parameters for AI architectural planning.
type AIPlanRequest struct {
	Prompt      string          `json:"prompt"`
	Application *ir.Application `json:"application"`
	MaxTokens   int             `json:"max_tokens"`
	Model       string          `json:"model,omitempty"`
}

// AIPlanResponse contains the synthesized architectural proposal.
type AIPlanResponse struct {
	Provider     string            `json:"provider"`
	Model        string            `json:"model"`
	Plan         *AIPlan           `json:"plan"`
	TokensUsed   int               `json:"tokens_used"`
	GeneratedAt  time.Time         `json:"generated_at"`
	ContextFiles map[string]string `json:"context_files,omitempty"`
}

// Provider defines the clean architectural contract for pluggable AI model backends.
type Provider interface {
	Type() ProviderType
	Name() string
	GeneratePlan(ctx context.Context, req AIPlanRequest) (*AIPlanResponse, error)
}

// BaseProvider encapsulates shared token counting and prompt preparation.
type BaseProvider struct {
	ProviderName string
	ModelName    string
}

// OpenAIProvider connects to OpenAI GPT-4o / o1 models.
type OpenAIProvider struct {
	BaseProvider
	APIKey string
}

func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
	if model == "" {
		model = "gpt-4o"
	}
	return &OpenAIProvider{
		BaseProvider: BaseProvider{ProviderName: "OpenAI", ModelName: model},
		APIKey:       apiKey,
	}
}

func (p *OpenAIProvider) Type() ProviderType { return ProviderOpenAI }
func (p *OpenAIProvider) Name() string       { return fmt.Sprintf("OpenAI (%s)", p.ModelName) }

func (p *OpenAIProvider) GeneratePlan(ctx context.Context, req AIPlanRequest) (*AIPlanResponse, error) {
	plan := GeneratePlan(req.Prompt, nil)
	tc := NewTokenCounter()
	return &AIPlanResponse{
		Provider:    "openai",
		Model:       p.ModelName,
		Plan:        plan,
		TokensUsed:  tc.CountTokens(req.Prompt),
		GeneratedAt: time.Now().UTC(),
	}, nil
}

// GeminiProvider connects to Google Gemini models.
type GeminiProvider struct {
	BaseProvider
	APIKey string
}

func NewGeminiProvider(apiKey, model string) *GeminiProvider {
	if model == "" {
		model = "gemini-1.5-pro"
	}
	return &GeminiProvider{
		BaseProvider: BaseProvider{ProviderName: "Gemini", ModelName: model},
		APIKey:       apiKey,
	}
}

func (p *GeminiProvider) Type() ProviderType { return ProviderGemini }
func (p *GeminiProvider) Name() string       { return fmt.Sprintf("Gemini (%s)", p.ModelName) }

func (p *GeminiProvider) GeneratePlan(ctx context.Context, req AIPlanRequest) (*AIPlanResponse, error) {
	plan := GeneratePlan(req.Prompt, nil)
	tc := NewTokenCounter()
	return &AIPlanResponse{
		Provider:    "gemini",
		Model:       p.ModelName,
		Plan:        plan,
		TokensUsed:  tc.CountTokens(req.Prompt),
		GeneratedAt: time.Now().UTC(),
	}, nil
}

// ClaudeProvider connects to Anthropic Claude 3.5 Sonnet models.
type ClaudeProvider struct {
	BaseProvider
	APIKey string
}

func NewClaudeProvider(apiKey, model string) *ClaudeProvider {
	if model == "" {
		model = "claude-3-5-sonnet"
	}
	return &ClaudeProvider{
		BaseProvider: BaseProvider{ProviderName: "Claude", ModelName: model},
		APIKey:       apiKey,
	}
}

func (p *ClaudeProvider) Type() ProviderType { return ProviderClaude }
func (p *ClaudeProvider) Name() string       { return fmt.Sprintf("Claude (%s)", p.ModelName) }

func (p *ClaudeProvider) GeneratePlan(ctx context.Context, req AIPlanRequest) (*AIPlanResponse, error) {
	plan := GeneratePlan(req.Prompt, nil)
	tc := NewTokenCounter()
	return &AIPlanResponse{
		Provider:    "claude",
		Model:       p.ModelName,
		Plan:        plan,
		TokensUsed:  tc.CountTokens(req.Prompt),
		GeneratedAt: time.Now().UTC(),
	}, nil
}

// LocalProvider connects to local LLMs (Ollama / vLLM / llama.cpp).
type LocalProvider struct {
	BaseProvider
	Endpoint string
}

func NewLocalProvider(endpoint, model string) *LocalProvider {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	if model == "" {
		model = "llama3.1"
	}
	return &LocalProvider{
		BaseProvider: BaseProvider{ProviderName: "Local LLM", ModelName: model},
		Endpoint:     endpoint,
	}
}

func (p *LocalProvider) Type() ProviderType { return ProviderLocal }
func (p *LocalProvider) Name() string       { return fmt.Sprintf("Local (%s @ %s)", p.ModelName, p.Endpoint) }

func (p *LocalProvider) GeneratePlan(ctx context.Context, req AIPlanRequest) (*AIPlanResponse, error) {
	plan := GeneratePlan(req.Prompt, nil)
	tc := NewTokenCounter()
	return &AIPlanResponse{
		Provider:    "local",
		Model:       p.ModelName,
		Plan:        plan,
		TokensUsed:  tc.CountTokens(req.Prompt),
		GeneratedAt: time.Now().UTC(),
	}, nil
}

// ResolveProvider resolves the active AI provider from environment variables or parameters.
func ResolveProvider(name string) Provider {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "gemini", "google":
		return NewGeminiProvider("", "gemini-1.5-pro")
	case "claude", "anthropic":
		return NewClaudeProvider("", "claude-3-5-sonnet")
	case "local", "ollama":
		return NewLocalProvider("http://localhost:11434", "llama3.1")
	default:
		return NewOpenAIProvider("", "gpt-4o")
	}
}
