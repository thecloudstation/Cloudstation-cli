# Storage Client Usage Example

## Integration with HandleDeployRepository

To integrate the storage client into the deployment handlers for local upload support, you would modify the `HandleDeployRepository` function in `internal/dispatch/handlers.go` as follows:

```go
import (
    // ... existing imports ...
    "github.com/thecloudstation/cloudstation-orchestrator/pkg/storage"
)

func HandleDeployRepository(ctx context.Context, params DeployRepositoryParams, natsClient *nats.Client, wsClient *websocket.Client, logger hclog.Logger) error {
    // ... existing code ...

    // After creating workDir, check if this is a local upload deployment
    if params.SourceType == "local_upload" && params.SourceUrl != "" {
        // Use storage client to download and extract tarball
        fmt.Println("=== Phase: Download & Extract ===")
        fmt.Printf("Downloading source from storage...\n")
        logger.Info("Downloading uploaded source", "sourceUrl", params.SourceUrl)

        if wsClient != nil {
            wsClient.Write("=== Phase: Download & Extract ===\nDownloading uploaded source...\n", "stdout")
        }
        updateDeploymentStep(backendClient, params.DeploymentID, "local_upload", backend.StepDownload, backend.StatusInProgress, "", logger)

        // Download and extract the tarball
        if err := storage.DownloadAndExtract(params.SourceUrl, workDir, logger); err != nil {
            fmt.Fprintf(os.Stdout, "ERROR [download]: %v\n", err)
            os.Stdout.Sync()
            updateDeploymentStep(backendClient, params.DeploymentID, "local_upload", backend.StepDownload, backend.StatusFailed, err.Error(), logger)
            if wsClient != nil {
                wsClient.Write(fmt.Sprintf("ERROR: Failed to download source: %v\n", err), "stderr")
                wsClient.Flush()
            }
            publishFailure(natsClient, params, logger, err)
            return fmt.Errorf("failed to download and extract source: %w", err)
        }

        fmt.Printf("Source extracted successfully to: %s\n", workDir)
        if wsClient != nil {
            wsClient.Write("Source extracted successfully\n", "stdout")
        }
        updateDeploymentStep(backendClient, params.DeploymentID, "local_upload", backend.StepDownload, backend.StatusCompleted, "", logger)

    } else {
        // Existing git clone logic
        // ...
    }

    // Continue with rest of deployment...
}
```

## Required Parameter Updates

You would need to add these fields to `DeployRepositoryParams`:

```go
type DeployRepositoryParams struct {
    // ... existing fields ...

    // For local upload support
    SourceType string `json:"sourceType,omitempty"` // "git" or "local_upload"
    SourceUrl  string `json:"sourceUrl,omitempty"`  // URL to download the source tarball from
}
```

## Backend Step Updates

You may want to add a new deployment step for download operations:

```go
const (
    // ... existing steps ...
    StepDownload DeploymentStep = "download"
)
```

## Usage from CLI

The CLI would use this by:

1. Uploading the local directory as a tarball to storage (S3, etc.)
2. Getting a presigned URL for the uploaded tarball
3. Calling the orchestrator with `sourceType: "local_upload"` and the presigned URL
4. The orchestrator downloads and extracts the tarball, then proceeds with the build

This approach allows the same deployment pipeline to work with both Git repositories and local directory uploads.