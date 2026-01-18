package git

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/hashicorp/go-hclog"
)

// CloneOptions represents options for cloning a Git repository
type CloneOptions struct {
	Repository     string
	Branch         string
	Token          string
	Provider       Provider
	DestinationDir string
	Logger         hclog.Logger
	OutputWriter   io.Writer // Optional writer for streaming git output
}

// Clone clones a Git repository to the specified destination directory
func Clone(opts CloneOptions) error {
	if opts.Logger == nil {
		opts.Logger = hclog.Default()
	}

	// Build the Git URL with authentication
	gitURL := BuildAuthURL(opts.Repository, opts.Token, opts.Provider)

	// Build git clone command
	args := []string{"clone"}
	if opts.Branch != "" {
		args = append(args, "--branch", opts.Branch)
	}
	args = append(args, gitURL, opts.DestinationDir)

	// Execute git clone
	cmd := exec.Command("git", args...)

	// If OutputWriter is provided, stream output to it
	var output []byte
	var err error
	if opts.OutputWriter != nil {
		cmd.Stdout = opts.OutputWriter
		cmd.Stderr = opts.OutputWriter
		err = cmd.Run()
	} else {
		output, err = cmd.CombinedOutput()
	}

	if err != nil {
		// Redact token from error messages
		errorMsg := string(output)
		if opts.Token != "" {
			errorMsg = strings.ReplaceAll(errorMsg, opts.Token, "***REDACTED***")
		}
		return fmt.Errorf("failed to clone repository %s (branch: %s): %s: %w",
			opts.Repository, opts.Branch, errorMsg, err)
	}

	opts.Logger.Info("Successfully cloned repository",
		"repository", opts.Repository,
		"branch", opts.Branch,
		"destination", opts.DestinationDir)

	return nil
}
