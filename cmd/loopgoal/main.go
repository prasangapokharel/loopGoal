package main

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
	"loopgoal/internal/config"
	"loopgoal/internal/detect"
	"loopgoal/internal/git"
	"loopgoal/internal/loop"
	"loopgoal/internal/state"
	"loopgoal/internal/verify"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	var err error
	switch command {
	case "init":
		err = runInit(args)
	case "run":
		err = runLoop(args)
	case "status":
		err = runStatus(args)
	case "stop":
		err = runStop(args)
	case "help", "--help", "-h":
		printUsage()
		return
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`LoopGoal — Autonomous Development Supervisor

Usage:
  loopgoal <command> [arguments]

Commands:
  init      Initialize LoopGoal configuration in .loopgoal/
  run       Start the autonomous development loop
  status    Display current execution state
  stop      Request graceful termination of a running loop
  help      Show this help message`)
}

func runInit(args []string) error {
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

func runRun(args []string) error {
	return runLoop(args)
}

func runLoop(args []string) error {
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
	agentAdapter := agent.NewCommandAgent(cfg.Agent.Command, cfg.Agent.Args, workDir)

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

func runStatus(args []string) error {
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
	fmt.Printf("Last commit: %s\n", lastCommit)

	fmt.Printf("Started:     %s\n", st.StartedAt.Format(time.RFC3339))
	fmt.Printf("Updated:     %s\n", st.UpdatedAt.Format(time.RFC3339))
	return nil
}

func runStop(args []string) error {
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
	// On Unix, sending signal 0 checks if process exists
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}
