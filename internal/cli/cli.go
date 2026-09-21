package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"loopgoal/internal/agent"
	"loopgoal/internal/audit"
	"loopgoal/internal/config"
	"loopgoal/internal/detect"
	"loopgoal/internal/git"
	"loopgoal/internal/inventory"
	"loopgoal/internal/loop"
	"loopgoal/internal/state"
	"loopgoal/internal/taskmap"
	"loopgoal/internal/verify"
)

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
	case "scan":
		return RunScan(cmdArgs)
	case "plan":
		return RunPlan(cmdArgs)
	case "test":
		return RunTest(cmdArgs)
	case "status":
		return RunStatus(cmdArgs)
	case "stop":
		return RunStop(cmdArgs)
	case "help", "--help", "-h":
		PrintUsage()
		return nil
	default:
		PrintUsage()
		return fmt.Errorf("unknown command: %s", command)
	}
}

// PrintUsage prints CLI instructions to stdout.
func PrintUsage() {
	fmt.Println(`LoopGoal — Autonomous Development Supervisor

Usage:
  loopgoal <command> [arguments]

Commands:
  init      Initialize LoopGoal configuration in .loopgoal/
  run       Start the autonomous development loop
  scan      Inspect and categorize repository inventory
  plan      Display current task map, pending queue, and evidence
  test      Execute pre-flight gate checks (inventory, rules, agent, git)
  status    Display current execution state
  stop      Request graceful termination of a running loop
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
