package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/urfave/cli/v2"

	// Import builtin plugins to register them
	_ "github.com/thecloudstation/cloudstation-orchestrator/builtin/csdocker"
	_ "github.com/thecloudstation/cloudstation-orchestrator/builtin/docker"
	_ "github.com/thecloudstation/cloudstation-orchestrator/builtin/github"
	_ "github.com/thecloudstation/cloudstation-orchestrator/builtin/goreleaser"
	_ "github.com/thecloudstation/cloudstation-orchestrator/builtin/nixpacks"
	_ "github.com/thecloudstation/cloudstation-orchestrator/builtin/nomadpack"
	_ "github.com/thecloudstation/cloudstation-orchestrator/builtin/noop"
	_ "github.com/thecloudstation/cloudstation-orchestrator/builtin/railpack"

	// Import secret providers to register them
	_ "github.com/thecloudstation/cloudstation-orchestrator/pkg/secrets/vault"
)

var (
	// Build-time variables set via ldflags
	// Example: go build -ldflags "-X main.Version=1.0.0 -X main.DefaultAPIURL=https://api.example.com"
	Version        = "v2.0.1"
	DefaultAPIURL  = "https://api.cloud-station.io"
	DefaultAuthURL = "https://auth.cloud-station.io"
)

// reorderArgs moves flags after positional arguments to before them within subcommands
// This allows: cs service stop svc_test --dry-run
// To work like: cs service stop --dry-run svc_test
func reorderArgs(args []string) []string {
	if len(args) <= 1 {
		return args
	}

	// Known top-level commands (to preserve command path)
	commands := map[string]bool{
		"init": true, "build": true, "deploy": true, "up": true, "release": true,
		"template": true, "image": true, "service": true, "deployment": true,
		"login": true, "logout": true, "whoami": true, "link": true,
		"billing": true, "project": true, "runner": true, "dispatch": true,
		"help": true, "h": true,
	}

	// Known subcommands
	subcommands := map[string]bool{
		"list": true, "info": true, "deploy": true, "tags": true, "create": true,
		"stop": true, "start": true, "restart": true, "delete": true,
		"status": true, "cancel": true, "clear-queue": true,
		"plans": true, "subscribe": true, "invoices": true, "add-card": true,
		"agent": true,
	}

	result := make([]string, 0, len(args))
	result = append(result, args[0]) // Keep program name

	// Find where command path ends
	cmdPathEnd := 1
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			break
		}
		if commands[arg] || subcommands[arg] {
			result = append(result, arg)
			cmdPathEnd = i + 1
		} else {
			break
		}
	}

	// Now reorder the rest: flags before positional args
	var flags []string
	var positional []string
	skipNext := false

	// Known flags that take a value
	valuedFlags := map[string]bool{
		"--config": true, "-c": true,
		"--api-url": true, "--log-level": true,
		"--app": true, "--project": true, "--env": true,
		"--team": true, "--service": true,
		"--search": true, "--tags": true, "--sort": true,
		"--page": true, "--limit": true, "--page-size": true,
		"--status": true, "--visibility": true,
		"--name": true, "--definition": true,
		"--server-addr": true, "--token": true,
		// Build/deploy flags
		"--builder": true, "--tag": true, "--path": true,
		"--output": true, "--image": true, "--port": true,
		"--main": true, "--repo": true, "--ldflags": true,
		"--target": true,
	}

	for i := cmdPathEnd; i < len(args); i++ {
		arg := args[i]

		if skipNext {
			flags = append(flags, arg)
			skipNext = false
			continue
		}

		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			// Check if this flag takes a value
			if valuedFlags[arg] {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					skipNext = true
				}
			}
		} else {
			positional = append(positional, arg)
		}
	}

	// Reconstruct: command path + flags + positional
	result = append(result, flags...)
	result = append(result, positional...)

	return result
}

func main() {
	app := &cli.App{
		Name:                   "cs",
		Usage:                  "CloudStation Orchestrator - Minimal deployment orchestrator",
		Version:                Version,
		UseShortOptionHandling: true,
		EnableBashCompletion:   true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "cloudstation.hcl",
				Usage:   "Path to configuration file",
				EnvVars: []string{"CS_CONFIG"},
			},
			&cli.StringFlag{
				Name:    "log-level",
				Value:   "info",
				Usage:   "Log level (trace, debug, info, warn, error)",
				EnvVars: []string{"CS_LOG_LEVEL"},
			},
		},
		Commands: []*cli.Command{
			// Lifecycle commands
			initCommand(),
			buildCommand(),
			deployCommand(),
			upCommand(),
			releaseCommand(),

			// Template & Image commands
			templateCommand(),
			imageCommand(),
			serviceCommand(),
			deploymentCommand(),

			// Authentication commands
			loginCommand(),
			logoutCommand(),
			whoamiCommand(),
			linkCommand(),

			// Billing commands
			billingCommand(),

			// Project commands
			projectCommand(),

			// System commands
			runnerCommand(),
			dispatchCommand(),
		},
		Before: func(c *cli.Context) error {
			// Setup logger
			level := hclog.LevelFromString(c.String("log-level"))
			logger := hclog.New(&hclog.LoggerOptions{
				Name:  "cs",
				Level: level,
				Color: hclog.AutoColor,
			})
			hclog.SetDefault(logger)

			return nil
		},
	}

	// Reorder args to allow flags after positional arguments
	// e.g., "cs service stop svc_test --dry-run" works like "cs service stop --dry-run svc_test"
	args := reorderArgs(os.Args)

	if err := app.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
