package cli

import (
	"fmt"
	"os"

	"pogonian/internal/version"
)

var Version = version.Current

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

	case "version", "-v", "--version", "--v", "-V", "versions":
		return RunVersion(cmdArgs)

	case "help", "-h", "--help":
		PrintHelp()
		return nil

	case "dev", "run":
		return RunDev(cmdArgs)

	case "build":
		return RunBuild(cmdArgs)

	case "test":
		return RunTest(cmdArgs)

	case "check", "lint":
		return RunCheck(cmdArgs)

	case "tree":
		return RunTree(cmdArgs)

	case "db":
		return RunDB(cmdArgs)

	case "db:migrate":
		return RunDBMigrate(cmdArgs)

	case "db:rollback":
		return RunDBRollback(cmdArgs)

	case "db:status":
		return RunDBStatus(cmdArgs)

	case "db:new":
		return RunDBNew(cmdArgs)

	case "inspect":
		return RunInspect(cmdArgs)

	case "analyze":
		return RunAnalyze(cmdArgs)

	case "import":
		return RunImport(cmdArgs)

	case "migrate":
		return RunMigrate(cmdArgs)

	case "upgrade", "update":
		return RunUpgrade(cmdArgs)

	case "modules", "marketplace", "components":
		return RunListModules(cmdArgs)

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

	case "swagger", "openapi":
		return RunSwagger(cmdArgs)

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
	fmt.Println("\x1b[1mDEVELOPMENT & TESTING:\x1b[0m")
	fmt.Println("  \x1b[1;36mcreate\x1b[0m <name> [dir]    Scaffold a new Pogonian application")
	fmt.Println("  \x1b[1;36mdev\x1b[0m                    Start live dev server with soft navigation & DevTools")
	fmt.Println("  \x1b[1;36mswagger\x1b[0m                Interactive Swagger UI and OpenAPI 3.0 specification")
	fmt.Println("  \x1b[1;36mbuild\x1b[0m [--standalone]   Compile and package optimized production release")
	fmt.Println("  \x1b[1;36mtest\x1b[0m [--e2e]           Run unified test orchestrator (Architecture, Security, Unit, E2E)")
	fmt.Println("  \x1b[1;36mcheck\x1b[0m                  Validate architecture rules, boundaries, and routes")
	fmt.Println("  \x1b[1;36mtree\x1b[0m [app]             Generate colorized application, routing, and component tree")
	fmt.Println("  \x1b[1;36mdoctor\x1b[0m                 Run full system, database, and project diagnostics")
	fmt.Println("  \x1b[1;36mclean\x1b[0m                  Clean build caches and temporary artifacts")
	fmt.Println("  \x1b[1;36mreset\x1b[0m                  Reset development database and rebuild project state")
	fmt.Println("  \x1b[1;36mupgrade\x1b[0m / \x1b[1;36mupdate\x1b[0m       Upgrade project configurations, .ai schemas, and runtime")
	fmt.Println()
	fmt.Println("\x1b[1mMARKETPLACE & MODULES (SHADCN-STYLE):\x1b[0m")
	fmt.Println("  \x1b[1;36mmodules\x1b[0m                List all marketplace UI components & full-stack modules")
	fmt.Println("  \x1b[1;36madd module\x1b[0m <name>      Install component (footer, header, sidebar, login, etc.)")
	fmt.Println("  \x1b[1;36madd component\x1b[0m <name>   Install UI-only component into app/components/")
	fmt.Println("  \x1b[1;36madd table\x1b[0m <name>       Scaffold full-stack Go backend & HTML frontend for entity")
	fmt.Println("  \x1b[1;36madd route\x1b[0m <path>       Add a new standalone filesystem route page")
	fmt.Println("  \x1b[1;36mremove module\x1b[0m <name>   Remove module or component from project")
	fmt.Println()
	fmt.Println("\x1b[1mDATABASE & MIGRATIONS:\x1b[0m")
	fmt.Println("  \x1b[1;36mdb:migrate\x1b[0m             Apply all pending schema migrations")
	fmt.Println("  \x1b[1;36mdb:rollback\x1b[0m [steps]    Revert the last batch of applied migrations")
	fmt.Println("  \x1b[1;36mdb:status\x1b[0m              Display migration status and version history")
	fmt.Println("  \x1b[1;36mdb:new\x1b[0m <name>          Create a new versioned .up.sql and .down.sql migration pair")
	fmt.Println()
	fmt.Println("\x1b[1mARCHITECTURE & IMPACT:\x1b[0m")
	fmt.Println("  \x1b[1;36mimpact\x1b[0m <target>        Analyze blast radius & affected layers of a change")
	fmt.Println("  \x1b[1;36mexplain\x1b[0m [target]       Explain full-stack architecture or specific feature flow")
	fmt.Println("  \x1b[1;36mblueprint\x1b[0m              Export architecture blueprint (Mermaid / JSON / YAML)")
	fmt.Println("  \x1b[1;36mdocs\x1b[0m                   Generate automatic API and route documentation")
	fmt.Println("  \x1b[1;36minspect\x1b[0m                Interactive architectural layer inspector")
	fmt.Println("  \x1b[1;36manalyze\x1b[0m                Extract Pogonian IR and dependency graph")
	fmt.Println()
	fmt.Println("\x1b[1mAI & MIGRATION:\x1b[0m")
	fmt.Println("  \x1b[1;36mai\x1b[0m <prompt>             AI architecture planning and token-efficient context")
	fmt.Println("  \x1b[1;36mai rollback\x1b[0m [id]       Rollback AI-applied architectural modifications")
	fmt.Println("  \x1b[1;36mimport\x1b[0m <path>          Analyze legacy React, Next, Vue, or Express app")
	fmt.Println("  \x1b[1;36mmigrate\x1b[0m <path>         Convert legacy application into Pogonian")
	fmt.Println()
	fmt.Println("\x1b[1mSECURITY & RESEARCH:\x1b[0m")
	fmt.Println("  \x1b[1;36msecurity audit\x1b[0m         Run automated security scanner and vulnerability probe")
	fmt.Println("  \x1b[1;36mbenchmark\x1b[0m [--stress]   Run scale and micro-benchmarks (10k+ routes stress test)")
	fmt.Println()
	fmt.Println("\x1b[1mFLAGS:\x1b[0m")
	fmt.Println("  -h, --help             Show help documentation")
	fmt.Println("  -v, --version, --v     Show Pogonian version (use 'pogn version -list' to see all versions)")
	fmt.Println()
}

// ExitOnError prints an error and exits.
func ExitOnError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "\x1b[1;31mError:\x1b[0m %s\n", err)
		os.Exit(1)
	}
}
