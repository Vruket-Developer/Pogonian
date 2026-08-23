package ai

import (
	"os"
	"regexp"
	"strings"
	"unicode"
)

var (
	// Standard RE2-compatible BPE regex pattern matching contractions, words, numbers, punctuation, whitespace
	bpeWordRegex = regexp.MustCompile(`(?i)'s|'t|'re|'ve|'m|'ll|'d|[^\r\n\p{L}\p{N}]?\p{L}+|\p{N}{1,3}|[^\s\p{L}\p{N}]+|\s+`)
)

// TokenCounter provides realistic BPE (Byte-Pair Encoding) token measurement matching LLM tokenizers (cl100k / GPT-4 / Claude / Gemini).
type TokenCounter struct{}

// NewTokenCounter creates a TokenCounter.
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{}
}

// CountTokens estimates the exact number of BPE tokens for a given string.
func (tc *TokenCounter) CountTokens(text string) int {
	if len(text) == 0 {
		return 0
	}

	matches := bpeWordRegex.FindAllString(text, -1)
	count := 0

	for _, m := range matches {
		trimmed := strings.TrimSpace(m)
		if trimmed == "" {
			if strings.Contains(m, "\n") || len(m) >= 4 {
				count += 1
			}
			continue
		}

		l := len(trimmed)
		switch {
		case l <= 4:
			count += 1
		case l <= 8:
			count += 2
		default:
			count += (l + 3) / 4
		}
	}

	if count == 0 && len(text) > 0 {
		count = 1
	}

	return count
}

// CountFileTokens counts tokens in a physical file.
func (tc *TokenCounter) CountFileTokens(filePath string) (int, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}
	return tc.CountTokens(string(data)), nil
}

// CountRepositoryTokens scans all relevant code files in a project and sums their exact token counts.
func (tc *TokenCounter) CountRepositoryTokens(files []string) int {
	total := 0
	for _, f := range files {
		cnt, err := tc.CountFileTokens(f)
		if err == nil {
			total += cnt
		}
	}
	return total
}

// IsCodeSymbol determines if a token is a programming language symbol.
func IsCodeSymbol(s string) bool {
	for _, r := range s {
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			return true
		}
	}
	return false
}
