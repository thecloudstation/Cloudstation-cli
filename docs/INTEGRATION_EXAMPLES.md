# Integration Example for DetectProjectName

## How to integrate in commands.go

### For the init command:

```go
func initCommand() *cli.Command {
    return &cli.Command{
        Name:  "init",
        Usage: "Initialize a new CloudStation configuration file",
        Action: func(c *cli.Context) error {
            configPath := c.String("config")

            // Check if file already exists
            if _, err := os.Stat(configPath); err == nil {
                return fmt.Errorf("configuration file already exists: %s", configPath)
            }

            // Auto-detect project name
            projectName := config.DetectProjectName()

            // Create default configuration with detected name
            defaultConfig := fmt.Sprintf(`project = "%s"

app "web" {
  build {
    use = "noop"
  }

  deploy {
    use = "noop"
  }
}
`, projectName)

            if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
                return fmt.Errorf("failed to write configuration file: %w", err)
            }

            fmt.Printf("Created configuration file: %s\n", configPath)
            fmt.Printf("Project name: %s (auto-detected)\n", projectName)
            return nil
        },
    }
}
```

### For the up command (if needed for validation):

```go
// In the executeLocalBuild function, after loading the config:
cfg, err := config.LoadConfigFile(configPath)
if err != nil {
    return fmt.Errorf("failed to load configuration: %w", err)
}

// If project name is empty, auto-detect it
if cfg.Project == "" {
    cfg.Project = config.DetectProjectName()
    logger.Info("auto-detected project name", "name", cfg.Project)
}
```

## Testing the integration

```bash
# Test in a git repository
cd /some/git/repo
cs init
# Should detect the repo name from git remote

# Test in a non-git directory
cd /tmp/my-awesome-project
cs init
# Should use "my-awesome-project" as the project name
```