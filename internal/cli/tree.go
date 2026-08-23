package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TreeNode represents a file or directory in the project tree.
type TreeNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	IsDir    bool        `json:"is_dir"`
	RouteURL string      `json:"route_url,omitempty"`
	Children []*TreeNode `json:"children,omitempty"`
}

// RunTree executes `pogn tree` to display the interactive application and routing tree.
func RunTree(args []string) error {
	projectDir := "."
	appOnly := false
	jsonOutput := false
	maxDepth := 10

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "app" || arg == "--app-only" || arg == "-a":
			appOnly = true
		case arg == "--json" || arg == "-j":
			jsonOutput = true
		case strings.HasPrefix(arg, "--depth="):
			var d int
			if _, err := fmt.Sscanf(strings.TrimPrefix(arg, "--depth="), "%d", &d); err == nil && d > 0 {
				maxDepth = d
			}
		case !strings.HasPrefix(arg, "-"):
			projectDir = arg
		}
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	targetDir := absDir
	if appOnly {
		targetDir = filepath.Join(absDir, "app")
		if _, err := os.Stat(targetDir); os.IsNotExist(err) {
			return fmt.Errorf("app directory not found at: %s", targetDir)
		}
	}

	rootNode := buildTreeNode(targetDir, absDir, 0, maxDepth)

	if jsonOutput {
		b, err := json.MarshalIndent(rootNode, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}

	fmt.Println()
	fmt.Printf("🌳 \x1b[1;36mPogonian Application Structure Tree\x1b[0m\n")
	fmt.Printf("   Root: \x1b[1m%s\x1b[0m\n", absDir)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	printTree(rootNode, "", true)

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  \x1b[1;30mLegend: \x1b[1;36mRoutes/Pages\x1b[0m | \x1b[1;33mLayouts\x1b[0m | \x1b[1;34mComponents\x1b[0m | \x1b[1;32mGo Backend\x1b[0m | \x1b[1;35mMigrations\x1b[0m")
	fmt.Println()

	return nil
}

func buildTreeNode(dirPath, rootDir string, currentDepth, maxDepth int) *TreeNode {
	name := filepath.Base(dirPath)
	if dirPath == rootDir {
		name = filepath.Base(rootDir) + "/"
	}

	node := &TreeNode{
		Name:     name,
		Path:     dirPath,
		IsDir:    true,
		Children: make([]*TreeNode, 0),
	}

	if currentDepth >= maxDepth {
		return node
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return node
	}

	for _, entry := range entries {
		entryName := entry.Name()

		// Skip hidden git, node_modules, dist, bin, caches
		if entryName == ".git" || entryName == "node_modules" || entryName == "dist" ||
			entryName == "bin" || entryName == ".pogonian_sandbox" {
			continue
		}

		childPath := filepath.Join(dirPath, entryName)

		if entry.IsDir() {
			childNode := buildTreeNode(childPath, rootDir, currentDepth+1, maxDepth)
			node.Children = append(node.Children, childNode)
		} else {
			routeURL := ""
			relApp, err := filepath.Rel(filepath.Join(rootDir, "app"), childPath)
			if err == nil && !strings.HasPrefix(relApp, "..") && strings.HasSuffix(entryName, "page.html") {
				routeURL = deriveRouteURL(relApp)
			}

			node.Children = append(node.Children, &TreeNode{
				Name:     entryName,
				Path:     childPath,
				IsDir:    false,
				RouteURL: routeURL,
			})
		}
	}

	return node
}

func deriveRouteURL(relAppPath string) string {
	dir := filepath.Dir(relAppPath)
	if dir == "." || dir == "" {
		return "/"
	}
	parts := strings.Split(dir, string(filepath.Separator))
	var urlParts []string
	for _, p := range parts {
		if strings.HasPrefix(p, "[") && strings.HasSuffix(p, "]") {
			urlParts = append(urlParts, ":"+p[1:len(p)-1])
		} else {
			urlParts = append(urlParts, p)
		}
	}
	return "/" + strings.Join(urlParts, "/")
}

func printTree(node *TreeNode, prefix string, isTail bool) {
	connector := "├── "
	if isTail {
		connector = "└── "
	}

	// Format node label with colors
	var label string
	if node.IsDir {
		label = fmt.Sprintf("\x1b[1m%s\x1b[0m/", node.Name)
		if node.Name == "app" || node.Name == "app/" {
			label = "\x1b[1;36mapp/\x1b[0m \x1b[1;30m(Filesystem Router)\x1b[0m"
		} else if node.Name == "components" || node.Name == "components/" {
			label = "\x1b[1;34mcomponents/\x1b[0m \x1b[1;30m(UI Registry)\x1b[0m"
		} else if node.Name == "server" || node.Name == "server/" {
			label = "\x1b[1;32mserver/\x1b[0m \x1b[1;30m(5-Tier Go Backend)\x1b[0m"
		} else if node.Name == "migrations" || node.Name == "migrations/" {
			label = "\x1b[1;35mmigrations/\x1b[0m \x1b[1;30m(SQL Schema Versioning)\x1b[0m"
		} else if node.Name == ".pogonian" || node.Name == ".pogonian/" {
			label = "\x1b[1;35m.pogonian/\x1b[0m \x1b[1;30m(Architecture & Security)\x1b[0m"
		}
	} else {
		switch {
		case strings.HasSuffix(node.Name, "page.html"):
			label = fmt.Sprintf("\x1b[1;36m%s\x1b[0m", node.Name)
			if node.RouteURL != "" {
				label += fmt.Sprintf("  \x1b[1;32m➔  GET %s\x1b[0m", node.RouteURL)
			}
		case strings.HasSuffix(node.Name, "layout.html"):
			label = fmt.Sprintf("\x1b[1;33m%s\x1b[0m  \x1b[1;30m(Root/Nested Layout)\x1b[0m", node.Name)
		case strings.HasSuffix(node.Name, ".go"):
			label = fmt.Sprintf("\x1b[1;32m%s\x1b[0m", node.Name)
		case strings.HasSuffix(node.Name, ".sql"):
			label = fmt.Sprintf("\x1b[1;35m%s\x1b[0m", node.Name)
		case strings.HasSuffix(node.Name, ".css"):
			label = fmt.Sprintf("\x1b[1;35m%s\x1b[0m", node.Name)
		case strings.HasSuffix(node.Name, ".yaml") || strings.HasSuffix(node.Name, ".ai"):
			label = fmt.Sprintf("\x1b[1;33m%s\x1b[0m", node.Name)
		default:
			label = node.Name
		}
	}

	if prefix == "" {
		fmt.Printf("📁 %s\n", label)
	} else {
		fmt.Printf("%s%s%s\n", prefix, connector, label)
	}

	childPrefix := prefix
	if prefix == "" {
		childPrefix = "   "
	} else if isTail {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	for i, child := range node.Children {
		isChildTail := i == len(node.Children)-1
		printTree(child, childPrefix, isChildTail)
	}
}
