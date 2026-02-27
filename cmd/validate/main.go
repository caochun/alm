// validate is a CLI tool that parses and cross-validates ALM DSL files.
//
// Usage:
//
//	validate -arch <app-arch.yaml> -deploy <deploy-env.yaml> -pipelines <dir>
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/alm/dsl"
)

func main() {
	archFile    := flag.String("arch",      "", "path to AppArchitecture YAML file")
	deployFile  := flag.String("deploy",    "", "path to DeploymentEnv YAML file")
	pipelineDir := flag.String("pipelines", "", "directory containing AppPipeline YAML templates")
	flag.Parse()

	if *archFile == "" || *deployFile == "" || *pipelineDir == "" {
		fmt.Fprintln(os.Stderr, "Usage: validate -arch <app-arch.yaml> -deploy <deploy.yaml> -pipelines <dir>")
		os.Exit(1)
	}

	ok := true

	// ── Load pipeline templates ───────────────────────────────────────────────
	pipelines, err := dsl.LoadPipelinesFromDir(*pipelineDir)
	if err != nil {
		fatalf("loading pipelines: %v", err)
	}
	fmt.Printf("Pipelines loaded (%d):\n", len(pipelines))
	for name, p := range pipelines {
		fmt.Printf("  ✓ %s  deliverables: %v\n", name, p.Deliverables)
	}

	// ── Load app architecture ─────────────────────────────────────────────────
	arch, err := dsl.ParseAppArchitecture(*archFile)
	if err != nil {
		fatalf("loading app architecture: %v", err)
	}
	fmt.Printf("\nAppArchitecture: %s\n", arch.Name)
	if arch.Description != "" {
		fmt.Printf("  %s\n", arch.Description)
	}

	order, err := arch.TopologicalOrder()
	if err != nil {
		fatalf("computing deployment order: %v", err)
	}
	fmt.Printf("  Deployment order (%d services):\n", len(order))
	for i, svc := range order {
		deps := "none"
		if len(svc.DependsOn) > 0 {
			deps = fmt.Sprintf("%v", svc.DependsOn)
		}
		fmt.Printf("    %d. %-20s pipeline: %-30s depends_on: %s\n",
			i+1, svc.Name, svc.Pipeline, deps)
	}

	// ── Load deployment env ───────────────────────────────────────────────────
	env, err := dsl.ParseDeploymentEnv(*deployFile)
	if err != nil {
		fatalf("loading deployment env: %v", err)
	}
	fmt.Printf("\nDeploymentEnv: %s  (environment: %s)\n", env.Name, env.Environment)
	fmt.Printf("  Services   : %d\n", len(env.Services))
	fmt.Printf("  Infra deps : %d\n", len(env.Dependencies))
	fmt.Printf("  Bindings   : %d\n", len(env.Bindings))

	if env.Network != nil {
		fmt.Printf("  Ingress    : %d\n", len(env.Network.Ingress))
	}

	for _, svc := range env.Services {
		res := "-"
		if svc.Compute != nil && svc.Compute.Resources != nil {
			r := svc.Compute.Resources
			res = fmt.Sprintf("cpu=%s mem=%s replicas=%d", r.CPU, r.Memory, r.Replicas)
		}
		fmt.Printf("    %-20s accepts: %-15s compute: %s (%s)\n",
			svc.Name, svc.Accepts, svc.Compute.Type, res)
	}

	for _, dep := range env.Dependencies {
		res := "-"
		if dep.Resources != nil {
			r := dep.Resources
			res = fmt.Sprintf("cpu=%s mem=%s", r.CPU, r.Memory)
			if r.Storage != "" {
				res += " storage=" + r.Storage
			}
		}
		fmt.Printf("    %-20s type: %-20s resources: %s\n", dep.Name, dep.Type, res)
	}

	// ── Cross-model validation ────────────────────────────────────────────────
	fmt.Println("\nValidating...")
	errs := dsl.Validate(env, arch, pipelines)
	if len(errs) > 0 {
		ok = false
		fmt.Fprintf(os.Stderr, "  Validation errors:\n")
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "    ✗ %v\n", e)
		}
	} else {
		fmt.Println("  ✓ All checks passed")
	}

	if !ok {
		os.Exit(1)
	}
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
