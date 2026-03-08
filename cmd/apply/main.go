package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/alm/domain"
	"github.com/alm/engine"
)

func main() {
	workspace := flag.String("workspace", "./workspace", "workspace root directory")
	pipelinesDir := flag.String("pipelines", "dsl/templates", "pipeline templates directory")
	app := flag.String("app", "", "application name (required)")
	env := flag.String("env", "", "deployment environment (required)")
	dryRun := flag.Bool("dry-run", false, "generate plan only, do not execute")
	force := flag.Bool("force", false, "skip incremental detection, regenerate all steps")
	flag.Parse()

	if *app == "" || *env == "" {
		fmt.Fprintln(os.Stderr, "Usage: apply -app <name> -env <name> [-workspace dir] [-pipelines dir] [--dry-run] [--force]")
		os.Exit(1)
	}

	eng := engine.NewEngine(*workspace, *pipelinesDir)

	plan, err := eng.Plan(*app, *env, *force)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating plan: %v\n", err)
		os.Exit(1)
	}

	printPlan(plan)

	if *dryRun {
		fmt.Println("\n(dry-run mode — no changes applied)")
		return
	}

	// Check if there are any pending steps.
	counts := plan.CountByStatus()
	if counts[domain.StepPending] == 0 {
		fmt.Println("\nNothing to do — all steps are up to date.")
		return
	}

	fmt.Printf("\n%d step(s) to execute. Proceed? [y/N] ", counts[domain.StepPending])
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		fmt.Println("Aborted.")
		return
	}

	// Re-generate and execute the plan.
	ctx := context.Background()
	plan, err = eng.Apply(ctx, *app, *env, false, *force)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nExecution error: %v\n", err)
		printPlan(plan)
		os.Exit(1)
	}

	fmt.Println("\n── Execution Complete ──")
	printPlan(plan)
}

func printPlan(plan *domain.ExecutionPlan) {
	fmt.Printf("ExecutionPlan: %s / %s\n", plan.App, plan.Environment)
	fmt.Println("────────────────────────────────")

	for _, phase := range plan.Phases {
		pending := 0
		skip := 0
		for _, s := range phase.Steps {
			if s.Status == domain.StepPending {
				pending++
			} else if s.Status == domain.StepSkipped {
				skip++
			}
		}
		fmt.Printf("\nPhase: %s (%d steps)\n", phase.Name, len(phase.Steps))

		for _, step := range phase.Steps {
			tag := statusTag(step.Status)
			fmt.Printf("  %s %s", tag, step.Description)
			if step.Status == domain.StepSkipped && step.Error == "" {
				fmt.Print("  (unchanged)")
			}
			if step.Error != "" {
				fmt.Printf("  (%s)", step.Error)
			}
			fmt.Println()
		}
	}
}

func statusTag(s domain.StepStatus) string {
	switch s {
	case domain.StepPending:
		return "[plan]"
	case domain.StepRunning:
		return "[ .. ]"
	case domain.StepSucceeded:
		return "[ ok ]"
	case domain.StepFailed:
		return "[FAIL]"
	case domain.StepSkipped:
		return "[skip]"
	default:
		return "[????]"
	}
}
