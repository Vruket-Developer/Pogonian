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

	case "dev", "run":
		return RunDev(cmdArgs)

	case "build":
		return RunBuild(cmdArgs)

	case "check", "lint":
		return RunCheck(cmdArgs)

	case "inspect":
		return RunInspect(cmdArgs)

	case "analyze":
		return RunAnalyze(cmdArgs)

	case "import":
		return RunImport(cmdArgs)

	case "migrate":
		return RunMigrate(cmdArgs)

	case "add":
		return RunAdd(cmdArgs)

	case "remove", "rm":
		return RunRemove(cmdArgs)

	case "impact":
		return RunImpact(cmdArgs)

	case "doctor":
		return RunDoctor(cmdArgs)

	case "explain":
		return RunExplain(cmdArgs)

	case "blueprint":
		return RunBlueprint(cmdArgs)

	case "docs":
		return RunDocs(cmdArgs)

	case "clean":
		return RunClean(cmdArgs)

	case "reset":
		return RunReset(cmdArgs)

	case "secret", "key:generate":
		return RunSecret(cmdArgs)

	case "ai":
		return RunAI(cmdArgs)

	case "security":
		return RunSecurity(cmdArgs)

	case "benchmark":
		return RunBenchmark(cmdArgs)

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
	fmt.Println("\x1b[1mDEVELOPMENT & DIAGNOSTICS:\x1b[0m")
	fmt.Println("  \x1b[1;36mcreate\x1b[0m <name> [dir]    Scaffold a new Pogonian application")
	fmt.Println("  \x1b[1;36mdev\x1b[0m                    Start live dev server with soft navigation & DevTools")
	fmt.Println("  \x1b[1;36mcheck\x1b[0m                  Validate architecture rules, boundaries, and routes")
	fmt.Println("  \x1b[1;36mbuild\x1b[0m [--standalone]   Compile and package optimized production release")
	fmt.Println("  \x1b[1;36mdoctor\x1b[0m                 Run full system, database, and project diagnostics")
	fmt.Println("  \x1b[1;36msecret\x1b[0m                 Generate or rotate cryptographic application secret keys")
	fmt.Println("  \x1b[1;36mclean\x1b[0m                  Clean build caches and temporary artifacts")
	fmt.Println("  \x1b[1;36mreset\x1b[0m                  Reset development database and rebuild project state")
	fmt.Println()
	fmt.Println("\x1b[1mARCHITECTURE & IMPACT:\x1b[0m")
	fmt.Println("  \x1b[1;36mimpact\x1b[0m <target>        Analyze blast radius & affected layers of a change")
	fmt.Println("  \x1b[1;36mexplain\x1b[0m [target]       Explain full-stack architecture or specific feature flow")
	fmt.Println("  \x1b[1;36mblueprint\x1b[0m              Export architecture blueprint (Mermaid / JSON / YAML)")
	fmt.Println("  \x1b[1;36mdocs\x1b[0m                   Generate automatic API and route documentation")
	fmt.Println("  \x1b[1;36madd table\x1b[0m <name>       Scaffold full-stack Go backend & HTML frontend for entity")
	fmt.Println("  \x1b[1;36madd route\x1b[0m <path>       Add a new standalone filesystem route page")
	fmt.Println("  \x1b[1;36mremove table\x1b[0m <name>    Delete entity backend tiers and frontend route")
	fmt.Println("  \x1b[1;36mremove route\x1b[0m <path>    Delete a standalone route")
	fmt.Println("  \x1b[1;36minspect\x1b[0m                Interactive architectural layer inspector")
	fmt.Println("  \x1b[1;36manalyze\x1b[0m                Extract Pogonian IR and dependency graph")
	fmt.Println()
	fmt.Println("\x1b[1mAI & MIGRATION:\x1b[0m")
	fmt.Println("  \x1b[1;36mai\x1b[0m <prompt>             AI architecture planning and token-efficient context")
	fmt.Println("  \x1b[1;36mimport\x1b[0m <path>          Analyze legacy React, Next, Vue, or Express app")
	fmt.Println("  \x1b[1;36mmigrate\x1b[0m <path>         Convert legacy application into Pogonian")
	fmt.Println()
	fmt.Println("\x1b[1mSECURITY & RESEARCH:\x1b[0m")
	fmt.Println("  \x1b[1;36msecurity audit\x1b[0m         Run static security and vulnerability scanner")
	fmt.Println("  \x1b[1;36mbenchmark\x1b[0m              Run MSc empirical research evaluation benchmark")
	fmt.Println()
	fmt.Println("\x1b[1mFLAGS:\x1b[0m")
	fmt.Println("  -h, --help             Show help documentation")
	fmt.Println("  -v, --version          Show Pogonian version")
	fmt.Println()
}

// ExitOnError prints an error and exits.
func ExitOnError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "\x1b[1;31mError:\x1b[0m %s\n", err)
		os.Exit(1)
	}
}
