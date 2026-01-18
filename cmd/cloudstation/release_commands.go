package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/thecloudstation/cloudstation-orchestrator/builtin/github"
	"github.com/thecloudstation/cloudstation-orchestrator/builtin/goreleaser"
	"github.com/urfave/cli/v2"
)

func releaseCommand() *cli.Command {
	return &cli.Command{
		Name:      "release",
		Usage:     "Build and release to GitHub (all platforms)",
		ArgsUsage: "<version>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "repo",
				Usage:   "GitHub repository (owner/repo)",
				EnvVars: []string{"GITHUB_REPOSITORY"},
			},
			&cli.StringFlag{
				Name:    "token",
				Usage:   "GitHub token",
				EnvVars: []string{"GITHUB_TOKEN"},
			},
			&cli.StringFlag{
				Name:  "name",
				Usage: "Binary name (default: auto-detect from go.mod)",
			},
			&cli.StringFlag{
				Name:  "main",
				Usage: "Path to main package (default: .)",
				Value: ".",
			},
			&cli.StringFlag{
				Name:  "output",
				Usage: "Output directory",
				Value: "./dist",
			},
			&cli.StringSliceFlag{
				Name:  "target",
				Usage: "Build targets (default: linux/amd64,linux/arm64,darwin/amd64,darwin/arm64,windows/amd64)",
			},
			&cli.StringFlag{
				Name:  "ldflags",
				Usage: "Additional ldflags",
			},
			&cli.BoolFlag{
				Name:  "draft",
				Usage: "Create as draft release",
			},
			&cli.BoolFlag{
				Name:  "prerelease",
				Usage: "Mark as prerelease",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Build only, don't push to GitHub",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output as JSON",
			},
		},
		Action: func(c *cli.Context) error {
			// Validate version argument
			if c.NArg() < 1 {
				return fmt.Errorf("version required\n\nUsage: cs release <version>\n\nExample: cs release v1.0.0")
			}
			version := c.Args().First()

			// Ensure version starts with 'v'
			if !strings.HasPrefix(version, "v") {
				version = "v" + version
			}

			// Auto-detect binary name from go.mod if not provided
			binaryName := c.String("name")
			if binaryName == "" {
				binaryName = detectBinaryName()
				if binaryName == "" {
					return fmt.Errorf("could not detect binary name - use --name flag")
				}
			}

			// Get GitHub token
			token := c.String("token")
			if token == "" && !c.Bool("dry-run") {
				// Try gh auth token
				if out, err := exec.Command("gh", "auth", "token").Output(); err == nil {
					token = strings.TrimSpace(string(out))
				}
				if token == "" {
					return fmt.Errorf("GitHub token required\n\nSet GITHUB_TOKEN or run: gh auth login")
				}
			}

			// Get repository
			repo := c.String("repo")
			if repo == "" && !c.Bool("dry-run") {
				repo = detectGitHubRepo()
				if repo == "" {
					return fmt.Errorf("could not detect GitHub repository - use --repo flag")
				}
			}

			// Build targets
			targets := c.StringSlice("target")
			if len(targets) == 0 {
				targets = []string{
					"linux/amd64",
					"linux/arm64",
					"darwin/amd64",
					"darwin/arm64",
					"windows/amd64",
				}
			}

			// Setup logger
			logger := hclog.New(&hclog.LoggerOptions{
				Name:   "release",
				Level:  hclog.Info,
				Output: os.Stderr,
			})

			ctx := hclog.WithContext(context.Background(), logger)

			// Build ldflags
			ldflags := fmt.Sprintf("-X main.Version=%s", version)
			if extra := c.String("ldflags"); extra != "" {
				ldflags += " " + extra
			}
			// Add default API URLs if set in environment
			if apiURL := os.Getenv("CS_API_URL"); apiURL != "" {
				ldflags += fmt.Sprintf(" -X 'main.DefaultAPIURL=%s'", apiURL)
			}
			if authURL := os.Getenv("CS_AUTH_URL"); authURL != "" {
				ldflags += fmt.Sprintf(" -X 'main.DefaultAuthURL=%s'", authURL)
			}

			logger.Info("starting release", "version", version, "binary", binaryName)

			// Phase 1: Build
			logger.Info("building binaries", "targets", targets)

			builder := &goreleaser.Builder{}
			builder.ConfigSet(map[string]interface{}{
				"name":       binaryName,
				"path":       c.String("main"),
				"version":    version,
				"output_dir": c.String("output"),
				"targets":    targets,
				"ldflags":    ldflags,
			})

			art, err := builder.Build(ctx)
			if err != nil {
				return fmt.Errorf("build failed: %w", err)
			}

			logger.Info("build completed", "binaries", len(targets))

			// If dry-run, stop here
			if c.Bool("dry-run") {
				if c.Bool("json") {
					fmt.Printf(`{"success": true, "version": "%s", "binaries": %d, "dry_run": true}%s`,
						version, len(targets), "\n")
				} else {
					fmt.Printf("DRY RUN: Built %d binaries for %s\n", len(targets), version)
					fmt.Printf("Output: %s/\n", c.String("output"))
				}
				return nil
			}

			// Phase 2: Push to GitHub
			logger.Info("pushing to GitHub releases", "repo", repo, "tag", version)

			registry := &github.Registry{}
			registry.ConfigSet(map[string]interface{}{
				"repository":     repo,
				"token":          token,
				"tag_name":       version,
				"release_name":   fmt.Sprintf("%s %s", binaryName, version),
				"generate_notes": true,
				"checksums":      true,
				"draft":          c.Bool("draft"),
				"prerelease":     c.Bool("prerelease"),
			})

			ref, err := registry.Push(ctx, art)
			if err != nil {
				return fmt.Errorf("push failed: %w", err)
			}

			logger.Info("release completed", "url", ref.Digest)

			if c.Bool("json") {
				fmt.Printf(`{"success": true, "version": "%s", "url": "%s", "binaries": %d}%s`,
					version, ref.Digest, len(targets), "\n")
			} else {
				fmt.Printf("\n✓ Released %s %s\n", binaryName, version)
				fmt.Printf("  URL: %s\n", ref.Digest)
				fmt.Printf("  Binaries: %d platforms\n", len(targets))
			}

			return nil
		},
	}
}

// detectBinaryName tries to detect the binary name from go.mod
func detectBinaryName() string {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "module ") {
			module := strings.TrimPrefix(line, "module ")
			module = strings.TrimSpace(module)
			// Get last part of module path
			parts := strings.Split(module, "/")
			return parts[len(parts)-1]
		}
	}
	return ""
}

// detectGitHubRepo tries to detect the GitHub repository from git remote
func detectGitHubRepo() string {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}

	url := strings.TrimSpace(string(out))

	// Handle SSH format: git@github.com:owner/repo.git
	if strings.HasPrefix(url, "git@github.com:") {
		repo := strings.TrimPrefix(url, "git@github.com:")
		repo = strings.TrimSuffix(repo, ".git")
		return repo
	}

	// Handle HTTPS format: https://github.com/owner/repo.git
	if strings.Contains(url, "github.com/") {
		idx := strings.Index(url, "github.com/")
		repo := url[idx+len("github.com/"):]
		repo = strings.TrimSuffix(repo, ".git")
		return repo
	}

	return ""
}
