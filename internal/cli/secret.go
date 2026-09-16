package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pogonian/pkg/config"
)

// RunSecret executes `pogn secret` / `pogn key:generate` to generate or rotate cryptographic application keys.
func RunSecret(args []string) error {
	projectDir := "."
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			projectDir = arg
		}
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	newKey, err := config.GenerateSecretKey()
	if err != nil {
		return fmt.Errorf("failed to generate secret key: %w", err)
	}

	envPath := filepath.Join(absDir, ".env")
	if data, err := os.ReadFile(envPath); err == nil {
		lines := strings.Split(string(data), "\n")
		updated := false
		for i, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "APP_KEY=") {
				lines[i] = "APP_KEY=" + newKey
				updated = true
				break
			}
		}

		if !updated {
			lines = append(lines, "APP_KEY="+newKey)
		}

		if err := os.WriteFile(envPath, []byte(strings.Join(lines, "\n")), 0600); err == nil {
			fmt.Println()
			fmt.Printf("🔑 \x1b[1;32mGenerated new application encryption key!\x1b[0m\n")
			fmt.Printf("   • Key:       \x1b[1;36m%s\x1b[0m\n", newKey)
			fmt.Printf("   • Updated:   %s\n\n", envPath)
			return nil
		}
	}

	fmt.Println()
	fmt.Printf("🔑 \x1b[1;32mGenerated Application Secret Key:\x1b[0m\n")
	fmt.Printf("   \x1b[1;36m%s\x1b[0m\n\n", newKey)
	return nil
}
