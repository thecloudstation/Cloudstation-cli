package nats

import (
	"bytes"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

// LogWriter implements io.Writer and publishes logs to NATS
type LogWriter struct {
	client       *Client
	deploymentID string
	jobID        int
	serviceID    string
	ownerID      string
	output       string // "stdout" or "stderr"
	phase        string
	buffer       []byte
	sequence     int
	mu           sync.Mutex
	logger       hclog.Logger
}

// NewLogWriter creates a NATS-based log writer
func NewLogWriter(client *Client, deploymentID string, jobID int, serviceID, ownerID, output, phase string, logger hclog.Logger) *LogWriter {
	if logger == nil {
		logger = hclog.Default()
	}

	return &LogWriter{
		client:       client,
		deploymentID: deploymentID,
		jobID:        jobID,
		serviceID:    serviceID,
		ownerID:      ownerID,
		output:       output,
		phase:        phase,
		buffer:       []byte{},
		sequence:     0,
		logger:       logger,
	}
}

// Write implements io.Writer - buffers and publishes log content
func (w *LogWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Append incoming bytes to buffer
	w.buffer = append(w.buffer, p...)

	// Process complete lines (ending with newline)
	for {
		idx := bytes.IndexByte(w.buffer, '\n')
		if idx == -1 {
			// No complete line found, keep buffering
			break
		}

		// Extract line including newline
		line := string(w.buffer[:idx+1])
		w.buffer = w.buffer[idx+1:]

		// Publish line to NATS
		w.sequence++
		payload := BuildLogPayload{
			DeploymentID: w.deploymentID,
			JobID:        w.jobID,
			ServiceID:    w.serviceID,
			OwnerID:      w.ownerID,
			LogOutput:    w.output,
			Content:      line,
			Timestamp:    time.Now().UnixMilli(),
			Sequence:     w.sequence,
			Phase:        w.phase,
		}

		// Publish asynchronously to avoid blocking writes
		// Log errors but don't fail the write
		if w.client != nil {
			if err := w.client.PublishBuildLog(payload); err != nil {
				w.logger.Warn("Failed to publish build log", "error", err, "deployment", w.deploymentID)
			}
		}
	}

	// Return number of bytes consumed (all of them)
	return len(p), nil
}

// SetPhase updates the current build phase
func (w *LogWriter) SetPhase(phase string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.phase = phase
	w.logger.Debug("Build phase changed", "phase", phase, "deployment", w.deploymentID)
}

// Flush sends any buffered content immediately
func (w *LogWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Send remaining buffer contents if any
	if len(w.buffer) > 0 {
		w.sequence++
		payload := BuildLogPayload{
			DeploymentID: w.deploymentID,
			JobID:        w.jobID,
			ServiceID:    w.serviceID,
			OwnerID:      w.ownerID,
			LogOutput:    w.output,
			Content:      string(w.buffer),
			Timestamp:    time.Now().UnixMilli(),
			Sequence:     w.sequence,
			Phase:        w.phase,
		}

		if w.client != nil {
			if err := w.client.PublishBuildLog(payload); err != nil {
				w.logger.Error("Failed to flush build log", "error", err, "deployment", w.deploymentID)
				return err
			}
		}

		w.buffer = []byte{}
	}

	return nil
}

// Close stops the writer and sends final flush
func (w *LogWriter) Close() error {
	return w.Flush()
}
