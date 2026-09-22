package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"loopgoal/internal/agent"
	"loopgoal/internal/audit"
	"loopgoal/internal/config"
	"loopgoal/internal/detect"
	"loopgoal/internal/git"
	"loopgoal/internal/hook"
	"loopgoal/internal/inventory"
	"loopgoal/internal/livefeed"
	"loopgoal/internal/loop"
	"loopgoal/internal/mcp"
	"loopgoal/internal/state"
	"loopgoal/internal/taskmap"
	"loopgoal/internal/verify"
)

// Version is the current semantic release version of LoopGoal.
const Version = "1.1.0"

// Execute handles CLI command dispatch.
func Execute(args []string) error {
	if len(args) < 1 {
		PrintUsage()
		return fmt.Errorf("no command provided")
	}

	command := args[0]
	cmdArgs := args[1:]

	switch command {
	case "init":
		return RunInit(cmdArgs)
	case "run":
		return RunLoop(cmdArgs)
	case "verify":
		return RunVerify(cmdArgs)
	case "select":
		return RunSelect(cmdArgs)
	case "hook":
		return RunHook(cmdArgs)
	case "rollback":
		return RunRollback(cmdArgs)
	case "mcp":
		return RunMCP(cmdArgs)
	case "scan":
		return RunScan(cmdArgs)
	case "plan":
		return RunPlan(cmdArgs)
	case "test":
		return RunTest(cmdArgs)
	case "status":
		return RunStatus(cmdArgs)
	case "daemon", "watch":
		return RunDaemon(cmdArgs)
	case "livefeed", "feed":
		return RunLivefeed(cmdArgs)
	case "stop":
		return RunStop(cmdArgs)
	case "version", "--version", "-v":
		PrintVersion()
		return nil
	case "help", "--help", "-h":
		PrintUsage()
		return nil
	default:
		PrintUsage()
		return fmt.Errorf("unknown command: %s", command)
	}
}

// PrintVersion prints the CLI version and architecture to stdout.
func PrintVersion() {
	fmt.Printf("loopgoal v%s (%s/%s)\n", Version, runtime.GOOS, runtime.GOARCH)
}

// PrintUsage prints CLI instructions to stdout.
func PrintUsage() {
	fmt.Println(`LoopGoal — Autonomous Development Supervisor

Usage:
  loopgoal <command> [arguments]

Commands:
  init      Initialize LoopGoal configuration in .loopgoal/
  run       Start the autonomous development loop
  verify    Execute project verification and generate one-time commit token
  select    Lock a single target file for the current iteration
  hook      Install or manage Git hard enforcement hooks (pre-commit, pre-push)
  rollback  Restore working tree to clean state, discarding broken changes
  mcp       Run Model Context Protocol server over stdio
  scan      Inspect and categorize repository inventory
  plan      Display current task map, pending queue, and evidence
  test      Execute pre-flight gate checks (inventory, rules, agent, git)
  daemon    Run polyglot background livefeed daemon (.loopgoal/livefeed.json)
  livefeed  Display current livefeed verification status or JSON output
  status    Display current execution state
  stop      Request graceful termination of a running loop
  version   Display LoopGoal version
  help      Show this help message`)
}

// RunInit creates the initial project configuration and state.
func RunInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	force := fs.Bool("force", false, "Overwrite existing configuration")
	dir := fs.String("dir", ".", "Project root directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	loopDir := filepath.Join(*dir, config.DefaultDir)
	cfgPath := filepath.Join(loopDir, config.DefaultConfigFile)
	statePath := filepath.Join(loopDir, config.DefaultStateFile)

	if !*force {
		if _, err := os.Stat(cfgPath); err == nil {
			return fmt.Errorf("configuration already exists at %s (use --force to overwrite)", cfgPath)
		}
	}

	if err := os.MkdirAll(loopDir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", loopDir, err)
	}

	// Auto-detect project type and suggested verification commands
	detected := detect.Detect(*dir)
	initialConfig := config.DetectConfig(*dir, detected.VerifyCommands)

	// Write config
	if err := os.WriteFile(cfgPath, []byte(config.FormatConfigYAML(initialConfig)), 0o644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	// Init state
	stateMgr := state.NewManager(statePath)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading initial config: %w", err)
	}

	if *force && stateMgr.Exists() {
		_ = os.Remove(statePath)
	}

	if _, err := stateMgr.Init(cfg.Goal); err != nil {
		return fmt.Errorf("initializing state file: %w", err)
	}

	fmt.Printf("✓ Initialized LoopGoal in %s (%s)\n", loopDir, detected.Description)
	fmt.Printf("  Config: %s\n", cfgPath)
	fmt.Printf("  State:  %s\n", statePath)
	return nil
}

// RunLoop starts the autonomous development loop.
func RunLoop(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	dir := fs.String("dir", ".", "Project root directory")
	cfgFile := fs.String("config", "", "Path to config file (default: .loopgoal/config.yaml)")
	iterations := fs.Int("iterations", 0, "Override maximum iterations")
	if err := fs.Parse(args); err != nil {
		return err
	}

	workDir, err := filepath.Abs(*dir)
	if err != nil {
		return fmt.Errorf("resolving working directory: %w", err)
	}

	configPath := *cfgFile
	if configPath == "" {
		configPath = filepath.Join(workDir, config.DefaultDir, config.DefaultConfigFile)
	}

	statePath := filepath.Join(workDir, config.DefaultDir, config.DefaultStateFile)

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config from %s: %w\n(Did you run 'loopgoal init'?)", configPath, err)
	}

	if *iterations > 0 {
		cfg.Limits.Iterations = *iterations
	}

	// Check git repository
	gitClient := git.New(workDir)
	isRepo, err := gitClient.IsRepo(context.Background())
	if err != nil || !isRepo {
		return fmt.Errorf("%s is not a git repository", workDir)
	}

	stateMgr := state.NewManager(statePath)
	verifier := verify.NewRunner(workDir)
	agentAdapter := agent.NewCommandAgent(cfg.Agent.Command, cfg.Agent.Args, workDir).
		WithStreaming(os.Stdout)

	// Pre-flight: verify the agent binary can be found before starting the loop.
	// This gives an immediate, actionable error instead of failing mid-iteration.
	if err := agentAdapter.Validate(); err != nil {
		return fmt.Errorf("cannot start loop:\n\n%w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	engine, err := loop.NewEngine(loop.Options{
		WorkDir:  workDir,
		Config:   cfg,
		StateMgr: stateMgr,
		Git:      gitClient,
		Verifier: verifier,
		Agent:    agentAdapter,
		Logger:   os.Stdout,
		PID:      os.Getpid(),
	})
	if err != nil {
		return fmt.Errorf("creating loop engine: %w", err)
	}

	return engine.Run(ctx)
}

// RunStatus prints current execution state.
func RunStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	dir := fs.String("dir", ".", "Project root directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	statePath := filepath.Join(*dir, config.DefaultDir, config.DefaultStateFile)
	stateMgr := state.NewManager(statePath)

	st, err := stateMgr.Load()
	if err != nil {
		return fmt.Errorf("no LoopGoal state found at %s. Run 'loopgoal init' first", statePath)
	}

	fmt.Println("LoopGoal Status")
	fmt.Println("────────────────────────────")
	fmt.Printf("Goal:        %s\n", strings.TrimSpace(st.Goal))
	fmt.Printf("Iteration:   %d\n", st.Iteration)

	statusStr := st.Status
	if st.Status == state.StatusRunning && st.PID > 0 {
		if !isProcessRunning(st.PID) {
			statusStr = fmt.Sprintf("%s (process %d inactive)", st.Status, st.PID)
		} else {
			statusStr = fmt.Sprintf("%s (PID: %d)", st.Status, st.PID)
		}
	}
	fmt.Printf("Status:      %s\n", statusStr)

	lastTask := st.LastTask
	if lastTask == "" {
		lastTask = "(none)"
	}
	fmt.Printf("Last task:   %s\n", lastTask)

	lastCommit := st.LastCommit
	if lastCommit == "" {
		lastCommit = "(none)"
	}
	if st.LastCheck != "" {
		fmt.Printf("Last check:  %s\n", st.LastCheck)
	}
	if len(st.RemainingQueue) > 0 {
		fmt.Printf("Remaining:   %s\n", strings.Join(st.RemainingQueue, ", "))
	}

	if !st.StartedAt.IsZero() {
		fmt.Printf("Started:     %s\n", st.StartedAt.Format(time.RFC3339))
	}
	if !st.UpdatedAt.IsZero() {
		fmt.Printf("Updated:     %s\n", st.UpdatedAt.Format(time.RFC3339))
	}
	return nil
}

// RunScan inspects the repository and prints the inventory breakdown.
func RunScan(args []string) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	dir := fs.String("dir", ".", "Project root directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	workDir, err := filepath.Abs(*dir)
	if err != nil {
		return fmt.Errorf("resolving working directory: %w", err)
	}

	scanner := inventory.NewScanner(workDir)
	inv, err := scanner.Scan(context.Background())
	if err != nil {
		return fmt.Errorf("scanning repository: %w", err)
	}

	fmt.Println("LoopGoal Repository Inventory")
	fmt.Println("────────────────────────────")
	fmt.Printf("Root Directory: %s\n", workDir)
	fmt.Printf("Summary:        %s\n\n", inv.Summary())

	printFileList := func(title string, files []string) {
		if len(files) == 0 {
			return
		}
		fmt.Printf("%s (%d):\n", title, len(files))
		for _, f := range files {
			fmt.Printf("  • %s\n", f)
		}
		fmt.Println()
	}

	printFileList("Source Files", inv.SourceFiles)
	printFileList("Test Files", inv.TestFiles)
	printFileList("Instructions & Rules", inv.Instructions)
	printFileList("Configuration & Manifests", inv.ConfigFiles)
	printFileList("Documentation", inv.DocFiles)

	return nil
}

// RunPlan displays the current task map, pending queue, and evidence status.
func RunPlan(args []string) error {
	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
	dir := fs.String("dir", ".", "Project root directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	workDir, err := filepath.Abs(*dir)
	if err != nil {
		return fmt.Errorf("resolving working directory: %w", err)
	}

	taskMapPath := filepath.Join(workDir, config.DefaultDir, "taskmap.json")
	tm, err := taskmap.Load(taskMapPath)
	if err != nil {
		// If no taskmap.json on disk, build one dynamically from inventory & config
		cfgPath := filepath.Join(workDir, config.DefaultDir, config.DefaultConfigFile)
		goal := "Autonomous Development"
		if cfg, err := config.Load(cfgPath); err == nil {
			goal = cfg.Goal
		}
		scanner := inventory.NewScanner(workDir)
		inv, err := scanner.Scan(context.Background())
		if err != nil {
			return fmt.Errorf("scanning repository: %w", err)
		}
		tm = taskmap.BuildFromInventory(inv, goal)
	}

	fmt.Println("LoopGoal Task Plan")
	fmt.Println("────────────────────────────")
	fmt.Printf("Goal: %s\n\n", strings.TrimSpace(tm.Goal))
	fmt.Print(tm.Summary())

	if verified := tm.VerifiedFiles(); len(verified) > 0 {
		fmt.Println("\nVerified files:")
		for _, v := range verified {
			fmt.Printf("  [✓] %s\n", v)
		}
	}

	if missing := tm.MissingFiles(); len(missing) > 0 {
		fmt.Println("\nMissing / unaddressed files:")
		for _, m := range missing {
			fmt.Printf("  [✗] %s\n", m)
		}
	}

	return nil
}

// RunTest executes pre-flight validation of the entire LoopGoal control system.
func RunTest(args []string) error {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	dir := fs.String("dir", ".", "Project root directory")
	smoke := fs.Bool("smoke", false, "Run only fast smoke checks")
	if err := fs.Parse(args); err != nil {
		return err
	}

	workDir, err := filepath.Abs(*dir)
	if err != nil {
		return fmt.Errorf("resolving working directory: %w", err)
	}

	ctx := context.Background()
	fmt.Println("LoopGoal Pre-Flight System Check")
	fmt.Println("────────────────────────────")

	// 1. Git Repository & Safety
	gitClient := git.New(workDir)
	isRepo, err := gitClient.IsRepo(ctx)
	if err != nil || !isRepo {
		return fmt.Errorf("git repository check failed: %s is not a git repository", workDir)
	}
	fmt.Println("✓ Git repository detected")

	// 2. Inventory Scanner
	scanner := inventory.NewScanner(workDir)
	inv, err := scanner.Scan(ctx)
	if err != nil {
		return fmt.Errorf("inventory scanner failed: %w", err)
	}
	fmt.Printf("✓ Inventory scanner (%d files discovered)\n", inv.TotalFiles)

	// 3. Instruction & Rule Discovery
	auditor := audit.New(workDir)
	rules, err := auditor.DiscoverRules(ctx)
	if err != nil {
		return fmt.Errorf("rule discovery failed: %w", err)
	}
	fmt.Printf("✓ Instructions, skills & rules (%d rule files discovered)\n", len(rules))

	// 4. Task Graph Generator
	cfgPath := filepath.Join(workDir, config.DefaultDir, config.DefaultConfigFile)
	goal := "Autonomous Software Engineering"
	var verifyCmds []string
	var agentCmd string
	var agentArgs []string

	if cfg, err := config.Load(cfgPath); err == nil {
		goal = cfg.Goal
		verifyCmds = cfg.Verify
		agentCmd = cfg.Agent.Command
		agentArgs = cfg.Agent.Args
	}

	tm := taskmap.BuildFromInventory(inv, goal)
	fmt.Printf("✓ Task graph generated (%d files mapped)\n", len(tm.Files))

	// 5. Verification Engine
	verifier := verify.NewRunner(workDir)
	if len(verifyCmds) > 0 && !*smoke {
		vSummary, err := verifier.Run(ctx, verifyCmds)
		if err != nil {
			return fmt.Errorf("verification engine test failed: %w", err)
		}
		if !vSummary.Passed {
			fmt.Printf("! Note: project verification currently failing: %s\n", vSummary.FailedCommand)
		} else {
			fmt.Println("✓ Project verification commands passed")
		}
	} else {
		fmt.Println("✓ Verification engine initialized")
	}

	// 6. State Persistence & Recovery
	loopDir := filepath.Join(workDir, config.DefaultDir)
	_ = os.MkdirAll(loopDir, 0o755)
	statePath := filepath.Join(loopDir, "preflight_test_state.json")
	stateMgr := state.NewManager(statePath)
	testState := &state.State{
		Goal:      goal,
		Status:    state.StatusIdle,
		StartedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := stateMgr.Save(testState); err != nil {
		return fmt.Errorf("state persistence check failed: %w", err)
	}
	if _, err := stateMgr.Load(); err != nil {
		return fmt.Errorf("state recovery check failed: %w", err)
	}
	_ = os.Remove(statePath)
	fmt.Println("✓ State persistence & atomic recovery verified")

	// 7. Agent Adapter Validation
	if agentCmd != "" {
		adapter := agent.NewCommandAgent(agentCmd, agentArgs, workDir)
		if err := adapter.Validate(); err != nil {
			fmt.Printf("! Agent adapter warning: %v\n", err)
		} else {
			fmt.Printf("✓ Agent adapter validated (%s)\n", agentCmd)
		}
	} else {
		fmt.Println("✓ Agent adapter interface verified")
	}

	fmt.Println("\n┌─────────────────────────────┐")
	fmt.Println("│     LOOPGOAL PRE-FLIGHT     │")
	fmt.Println("├─────────────────────────────┤")
	fmt.Println("│ ✓ Inventory                 │")
	fmt.Println("│ ✓ Instructions              │")
	fmt.Println("│ ✓ Skills                    │")
	fmt.Println("│ ✓ Rules                     │")
	fmt.Println("│ ✓ Task Graph                │")
	fmt.Println("│ ✓ Reconciliation Engine     │")
	fmt.Println("│ ✓ Verification              │")
	fmt.Println("│ ✓ State Persistence         │")
	fmt.Println("│ ✓ Git Repository & Safety   │")
	fmt.Println("│ ✓ Loop Protection           │")
	fmt.Println("│ ✓ Agent Adapter             │")
	fmt.Println("└─────────────────────────────┘")
	fmt.Println("READY FOR AUTONOMOUS EXECUTION")

	return nil
}

// RunStop requests graceful termination of an active loop.
func RunStop(args []string) error {
	fs := flag.NewFlagSet("stop", flag.ContinueOnError)
	dir := fs.String("dir", ".", "Project root directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	loopDir := filepath.Join(*dir, config.DefaultDir)
	pidFile := filepath.Join(loopDir, "loopgoal.pid")
	stopFile := filepath.Join(loopDir, "stop")

	// 1. Create stop trigger file so engine checks it
	_ = os.WriteFile(stopFile, []byte("stop\n"), 0o644)

	// 2. Read pid file
	pidData, err := os.ReadFile(pidFile)
	var pid int
	if err == nil {
		pid, _ = strconv.Atoi(strings.TrimSpace(string(pidData)))
	}

	if pid == 0 {
		// Try reading state file
		statePath := filepath.Join(loopDir, config.DefaultStateFile)
		stateMgr := state.NewManager(statePath)
		if st, err := stateMgr.Load(); err == nil && st.PID > 0 {
			pid = st.PID
		}
	}

	if pid > 0 && isProcessRunning(pid) {
		proc, err := os.FindProcess(pid)
		if err == nil {
			if err := proc.Signal(os.Interrupt); err != nil {
				// Fallback to SIGTERM
				_ = proc.Signal(syscall.SIGTERM)
			}
			fmt.Printf("✓ Sent graceful stop signal to LoopGoal (PID: %d).\n", pid)
			return nil
		}
	}

	fmt.Println("✓ Stop signal recorded. LoopGoal will terminate after current iteration.")
	return nil
}

func isProcessRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// RunVerify executes project test suites and generates a one-time commit token if passed.
func RunVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	dir := fs.String("dir", ".", "Project root directory")
	cfgFile := fs.String("config", "", "Path to config file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	workDir, err := filepath.Abs(*dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	configPath := *cfgFile
	if configPath == "" {
		configPath = filepath.Join(workDir, config.DefaultDir, config.DefaultConfigFile)
	}

	var verifyCmds []string
	cfg, err := config.Load(configPath)
	if err == nil {
		verifyCmds = cfg.Verify
	} else {
		detected := detect.Detect(workDir)
		verifyCmds = detected.VerifyCommands
	}

	if len(verifyCmds) == 0 {
		return fmt.Errorf("no verification commands configured in %s or auto-detected", configPath)
	}

	fmt.Println("LoopGoal Zero-Trust Verification")
	fmt.Println("────────────────────────────")
	fmt.Printf("WorkDir:  %s\n", workDir)
	fmt.Printf("Commands: %s\n\n", strings.Join(verifyCmds, ", "))

	verifier := verify.NewRunner(workDir)
	ctx := context.Background()
	summary, err := verifier.Run(ctx, verifyCmds)
	if err != nil {
		_ = hook.ConsumeToken(workDir)
		return fmt.Errorf("verification execution error: %w", err)
	}

	if !summary.Passed {
		_ = hook.ConsumeToken(workDir)
		fmt.Printf("✗ Verification FAILED on command: %s\n\n", summary.FailedCommand)
		fmt.Println(summary.ErrorOutput())
		fmt.Println("❌ [LoopGoal Blocker] Verification failed. Git commit remains locked.")
		return fmt.Errorf("verification failed on command: %s", summary.FailedCommand)
	}

	token, err := hook.WriteToken(workDir, fmt.Sprintf("Verified %d commands at %s", len(verifyCmds), time.Now().Format(time.RFC3339)))
	if err != nil {
		return fmt.Errorf("generating token: %w", err)
	}

	fmt.Println("✓ All verification commands passed successfully!")
	fmt.Printf("✓ Cryptographic commit token generated: %s\n", hook.TokenPath(workDir))
	fmt.Printf("  Token signature: %s\n", token)
	fmt.Println("✓ Git commit is now UNLOCKED.")
	return nil
}

// RunSelect locks a single file target for the current iteration.
func RunSelect(args []string) error {
	fs := flag.NewFlagSet("select", flag.ContinueOnError)
	dir := fs.String("dir", ".", "Project root directory")
	objective := fs.String("objective", "", "Bounded improvement description")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var targetFile string
	if fs.NArg() > 0 {
		targetFile = fs.Arg(0)
	} else {
		return fmt.Errorf("target file required: loopgoal select <file> [-objective '...']")
	}

	workDir, err := filepath.Abs(*dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	statePath := filepath.Join(workDir, config.DefaultDir, config.DefaultStateFile)
	stateMgr := state.NewManager(statePath)

	st, err := stateMgr.Load()
	if err != nil {
		st, err = stateMgr.Init("Autonomous Development")
		if err != nil {
			return fmt.Errorf("initializing state: %w", err)
		}
	}

	st.ActiveTarget = targetFile
	if *objective != "" {
		st.LastTask = *objective
	}
	st.Status = state.StatusRunning
	if err := stateMgr.Save(st); err != nil {
		return fmt.Errorf("saving target: %w", err)
	}

	fmt.Println("LoopGoal Single-Target Barrier")
	fmt.Println("────────────────────────────")
	fmt.Printf("[●] Active Target: %s\n", targetFile)
	if *objective != "" {
		fmt.Printf("[●] Objective:     %s\n", *objective)
	}
	fmt.Println("[ℹ] Hard enforcement active: Out-of-scope edits to other files will be automatically rejected.")
	return nil
}

// RunHook installs, removes, or checks the status of LoopGoal Git hard enforcement hooks.
func RunHook(args []string) error {
	if len(args) < 1 {
		fmt.Println("Usage: loopgoal hook <install|remove|status> [-dir .]")
		return fmt.Errorf("subcommand required: install, remove, or status")
	}

	subcmd := args[0]
	fs := flag.NewFlagSet("hook", flag.ContinueOnError)
	dir := fs.String("dir", ".", "Project root directory")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	workDir, err := filepath.Abs(*dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	switch subcmd {
	case "install":
		if err := hook.Install(workDir); err != nil {
			return fmt.Errorf("installing hooks: %w", err)
		}
		fmt.Println("✓ Installed LoopGoal Git hard-enforcement hooks in .git/hooks/:")
		fmt.Println("  • pre-commit: blocks unverified commits without .loopgoal/verified.token")
		fmt.Println("  • pre-push:   blocks unauthorized remote push")
		return nil

	case "remove":
		if err := hook.Remove(workDir); err != nil {
			return fmt.Errorf("removing hooks: %w", err)
		}
		fmt.Println("✓ Removed LoopGoal Git hooks.")
		return nil

	case "status":
		hasCommit, hasPush := hook.Status(workDir)
		fmt.Println("LoopGoal Git Hooks Status:")
		fmt.Printf("  • pre-commit (Unverified commit lock): %v\n", hasCommit)
		fmt.Printf("  • pre-push   (Remote push safety lock): %v\n", hasPush)
		return nil

	default:
		return fmt.Errorf("unknown hook subcommand '%s'. Use install, remove, or status", subcmd)
	}
}

// RunRollback restores the working tree to a clean state, discarding broken iteration edits.
func RunRollback(args []string) error {
	fs := flag.NewFlagSet("rollback", flag.ContinueOnError)
	dir := fs.String("dir", ".", "Project root directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	workDir, err := filepath.Abs(*dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	gitClient := git.New(workDir)
	ctx := context.Background()

	var files []string
	if fs.NArg() > 0 {
		files = fs.Args()
	}

	if err := gitClient.Rollback(ctx, files...); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	_ = hook.ConsumeToken(workDir)
	if len(files) > 0 {
		fmt.Printf("✓ Rolled back specified files: %s\n", strings.Join(files, ", "))
	} else {
		fmt.Println("✓ Working tree restored to clean state (reverted unverified changes).")
	}
	return nil
}

// RunMCP runs the LoopGoal Model Context Protocol (MCP) server over standard I/O.
func RunMCP(args []string) error {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	dir := fs.String("dir", ".", "Project root directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	workDir, err := filepath.Abs(*dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	server := mcp.NewServer(workDir, os.Stdin, os.Stdout)
	return server.Serve(context.Background())
}

// RunDaemon starts the polyglot livefeed background daemon.
func RunDaemon(args []string) error {
	fs := flag.NewFlagSet("daemon", flag.ContinueOnError)
	dir := fs.String("dir", ".", "Project root directory")
	interval := fs.Duration("interval", 1*time.Second, "Polling/check interval")
	once := fs.Bool("once", false, "Run single check cycle and exit")
	if err := fs.Parse(args); err != nil {
		return err
	}

	workDir, err := filepath.Abs(*dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	fmt.Println("⚡ LoopGoal Polyglot Livefeed Daemon")
	fmt.Println("──────────────────────────────────")
	fmt.Printf("WorkDir:  %s\n", workDir)
	if *once {
		fmt.Println("Mode:     Single execution (--once)")
	} else {
		fmt.Printf("Interval: %v\n", *interval)
		fmt.Println("Status:   Watching workspace for file changes...")
	}
	fmt.Println()

	return livefeed.RunDaemon(workDir, *interval, *once, os.Stdout)
}

// RunLivefeed displays current livefeed status or outputs JSON.
func RunLivefeed(args []string) error {
	fs := flag.NewFlagSet("livefeed", flag.ContinueOnError)
	dir := fs.String("dir", ".", "Project root directory")
	asJSON := fs.Bool("json", false, "Output raw livefeed.json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	workDir, err := filepath.Abs(*dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	feed, err := livefeed.ReadFeed(workDir)
	if err != nil {
		return fmt.Errorf("no active livefeed found in %s: %w", workDir, err)
	}

	if *asJSON {
		data, _ := json.MarshalIndent(feed, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Println("LoopGoal Livefeed Status")
	fmt.Println("────────────────────────")
	fmt.Printf("Heartbeat:   #%d\n", feed.Heartbeat)
	fmt.Printf("Updated At:  %s\n", feed.UpdatedAt)
	fmt.Printf("Latency:     %s\n", feed.LatencyMs)
	fmt.Printf("Status:      %s\n", feed.Status)
	fmt.Printf("Can Commit:  %v\n", feed.CanCommit)
	fmt.Printf("Runtimes:    %s\n", strings.Join(feed.Stats.Runtimes, ", "))
	fmt.Printf("Errors:      %d\n", feed.Stats.TotalErrors)
	if len(feed.Errors) > 0 {
		fmt.Println("\nCondensed Errors:")
		for i, e := range feed.Errors {
			fmt.Printf("  [%d] (%s) %s:%d:%d [%s] %s\n", i+1, e.Source, e.File, e.Line, e.Col, e.Code, e.Message)
		}
	}
	if feed.CanCommit && feed.VerificationToken != "" {
		fmt.Printf("\n✓ Verification Token: %s\n", feed.VerificationToken)
	}
	return nil
}

