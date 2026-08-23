package ai_test

import (
	"os"
	"path/filepath"
	"testing"

	"pogonian/internal/ai"
)

func TestTokenizer_BPEAccuracy(t *testing.T) {
	counter := ai.NewTokenCounter()

	// 1. Basic text
	text := "Hello world! This is Pogonian framework."
	tokens := counter.CountTokens(text)
	if tokens < 6 || tokens > 16 {
		t.Errorf("Expected ~10-13 tokens for basic sentence, got %d", tokens)
	}

	// 2. Code snippet
	code := `func (s *Server) Start() error {
		return s.httpServer.ListenAndServe()
	}`
	codeTokens := counter.CountTokens(code)
	if codeTokens < 10 || codeTokens > 30 {
		t.Errorf("Expected ~15-25 tokens for Go function, got %d", codeTokens)
	}

	// 3. File counting
	tempDir, err := os.MkdirTemp("", "pogn_tok_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "sample.go")
	_ = os.WriteFile(filePath, []byte(code), 0644)

	fileTokens, err := counter.CountFileTokens(filePath)
	if err != nil || fileTokens != codeTokens {
		t.Errorf("CountFileTokens mismatch: got %d, expected %d", fileTokens, codeTokens)
	}
}
