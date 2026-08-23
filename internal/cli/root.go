package cli

import (
	"fmt"
	"os"
)

const Version = "0.1.0-alpha"

const banner = "\n" +
	" \x1b[1;36m___  __  ___  __  _  _ _  __  _  _ \x1b[0m\n" +
	" \x1b[1;36m| o \\/  \\/ _/ /__\\| \\| | |/  \\| \\| |\x1b[0m\n" +
	" \x1b[1;36m|  _/ /\\ \\ \\_| \\/ | \\\\ | | /\\ | \\\\ |\x1b[0m\n" +
	" \x1b[1;36m|_|  \\__/ \\__/\\__/|_|\\_|_|_||_|_|\\_|\x1b[0m\n" +
	"\x1b[1;30mHTML-First Full-Stack Web Framework & AI Platform\x1b[0m\n"

// Run executes the root CLI dispatcher.
func Run(args []string) error {
	if len(args) == 0 {
		PrintHelp()
		return nil
	}

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "create", "new", "init":
		opts, err := ExtractCreateOptions(cmdArgs)
		if err != nil {
			return err
		}
		return RunCreate(opts)

	case "version", "-v", "--version":
		fmt.Printf("Pogonian CLI v%s\n", Version)
		return nil

	case "help", "-h", "--help":
		PrintHelp()
		return nil

	case "dev":
		fmt.Println("⚡ Starting Pogonian development server (Coming in Step 08)...")
		return nil

	case "build":
		fmt.Println("📦 Compiling Pogonian application (Coming in Step 22)...")
		return nil

	case "check":
		fmt.Println("🔍 Running Pogonian architecture and route validator (Coming in Step 18)...")
		return nil

	case "inspect":
		fmt.Println("🔎 Inspecting project architecture and routes (Coming in Step 17)...")
		return nil

	case "analyze":
		fmt.Println("📊 Analyzing project dependency graph (Coming in Step 17)...")
		return nil

	case "import":
		fmt.Println("📥 Importing legacy framework application (Coming in Step 22)...")
		return nil

	case "migrate":
		fmt.Println("🔄 Running Pogonian migration engine (Coming in Step 22)...")
		return nil

	case "ai":
		fmt.Println("🧠 Pogonian AI Context Engine (Coming in Step 20)...")
		return nil

	case "security":
		fmt.Println("🛡️  Pogonian Security Audit (Coming in Step 21)...")
		return nil

	default:
		return fmt.Errorf("unknown command '%s'. Run 'pogn --help' for usage", cmd)
	}
}

// PrintHelp prints the CLI help message.
func PrintHelp() {
	fmt.Print(banner)
	fmt.Println()
	fmt.Println("\x1b[1mUSAGE:\x1b[0m")
	fmt.Println("  pogn <command> [arguments] [flags]")
	fmt.Println()
	fmt.Println("\x1b[1mCORE COMMANDS:\x1b[0m")
	fmt.Println("  \x1b[1;36mcreate\x1b[0m <name>          Scaffold a new Pogonian application")
	fmt.Println("  \x1b[1;36mdev\x1b[0m                    Start development server with soft navigation and live reload")
	fmt.Println("  \x1b[1;36mbuild\x1b[0m                  Compile HTML, CSS, assets, and Go backend into binary")
	fmt.Println("  \x1b[1;36mstart\x1b[0m                  Start production server")
	fmt.Println()
	fmt.Println("\x1b[1mARCHITECTURE & AI:\x1b[0m")
	fmt.Println("  \x1b[1;36mcheck\x1b[0m                  Validate architecture rules, boundaries, and routes")
	fmt.Println("  \x1b[1;36minspect\x1b[0m                Inspect route graph, middleware, and backend layers")
	fmt.Println("  \x1b[1;36manalyze\x1b[0m                Extract Pogonian IR and dependency graph")
	fmt.Println("  \x1b[1;36mai\x1b[0m <prompt>             AI-assisted architecture planning and token-efficient context")
	fmt.Println()
	fmt.Println("\x1b[1mMIGRATION & SECURITY:\x1b[0m")
	fmt.Println("  \x1b[1;36mimport\x1b[0m <path>          Analyze and import legacy React, Next, Vue, or backend apps")
	fmt.Println("  \x1b[1;36mmigrate\x1b[0m <path>         Execute progressive framework migration to Pogonian IR")
	fmt.Println("  \x1b[1;36msecurity audit\x1b[0m         Run full-stack security and vulnerability analysis")
	fmt.Println()
	fmt.Println("\x1b[1mFLAGS:\x1b[0m")
	fmt.Println("  -h, --help             Show help documentation")
	fmt.Println("  -v, --version          Show Pogonian version")
	fmt.Println()
	fmt.Println("Run '\x1b[1;36mpogn <command> --help\x1b[0m' for more information about a specific command.")
	fmt.Println()
}

// ExitOnError prints an error and exits.
func ExitOnError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "\x1b[1;31mError:\x1b[0m %s\n", err)
		os.Exit(1)
	}
}
