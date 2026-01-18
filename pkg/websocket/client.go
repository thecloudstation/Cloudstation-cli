package websocket

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	socketio "github.com/zishang520/socket.io/clients/socket/v3"
)

// Client handles WebSocket connections for streaming build logs
type Client struct {
	socket       *socketio.Socket
	deploymentID string
	stdoutBuffer []string
	stderrBuffer []string
	counter      int
	connected    bool
	mu           sync.Mutex
	done         chan struct{}
	logger       hclog.Logger
}

// NewClient creates a new WebSocket client and connects to the build-logs endpoint
func NewClient(wsURL, deploymentID string, logger hclog.Logger) (*Client, error) {
	client := &Client{
		deploymentID: deploymentID,
		stdoutBuffer: []string{},
		stderrBuffer: []string{},
		counter:      0,
		connected:    false,
		done:         make(chan struct{}),
		logger:       logger,
	}

	// Parse and append /build-logs path
	u, err := url.Parse(wsURL)
	if err != nil {
		logger.Warn("Failed to parse WebSocket URL", "url", wsURL, "error", err)
		return client, err
	}
	u.Path = "/build-logs"

	// Create Socket.IO client
	socket, err := socketio.Connect(u.String(), nil)
	if err != nil {
		logger.Warn("Failed to connect to Socket.IO", "url", u.String(), "error", err)
		return client, err
	}

	// Set up event handlers
	socket.On("connect", func(args ...any) {
		client.mu.Lock()
		client.connected = true
		client.mu.Unlock()
		logger.Info("Socket.IO connected successfully", "url", u.String(), "deploymentID", deploymentID)
	})

	socket.On("disconnect", func(args ...any) {
		client.mu.Lock()
		client.connected = false
		client.mu.Unlock()
		logger.Info("Socket.IO disconnected", "deploymentID", deploymentID)
	})

	socket.On("connect_error", func(args ...any) {
		if len(args) > 0 {
			logger.Warn("Socket.IO connection error", "error", args[0])
		}
	})

	client.socket = socket
	client.connected = socket.Connected()

	// Start background flush goroutine
	go client.flushLoop()

	return client, nil
}

// Write adds a log chunk to the appropriate buffer
func (c *Client) Write(chunk string, output string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if output == "stdout" {
		c.stdoutBuffer = append(c.stdoutBuffer, chunk)
	} else if output == "stderr" {
		c.stderrBuffer = append(c.stderrBuffer, chunk)
	}
}

// flushLoop runs in a goroutine and flushes logs every 500ms
func (c *Client) flushLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.flush()
		case <-c.done:
			return
		}
	}
}

// flush sends buffered logs to WebSocket
func (c *Client) flush() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return
	}

	// Flush stdout buffer
	if len(c.stdoutBuffer) > 0 {
		content := fmt.Sprintf("%s\nstdout\n%d\n%s",
			c.deploymentID,
			c.counter,
			strings.Join(c.stdoutBuffer, "\n"))

		c.socket.Emit("build-log", content)
		c.stdoutBuffer = []string{}
		c.counter++
	}

	// Flush stderr buffer
	if len(c.stderrBuffer) > 0 {
		content := fmt.Sprintf("%s\nstderr\n%d\n%s",
			c.deploymentID,
			c.counter,
			strings.Join(c.stderrBuffer, "\n"))

		c.socket.Emit("build-log", content)
		c.stderrBuffer = []string{}
		c.counter++
	}
}

// Flush forces immediate flush of all buffered logs
func (c *Client) Flush() error {
	c.flush()
	return nil
}

// End closes the WebSocket connection and stops the flush loop
func (c *Client) End() error {
	// Final flush
	c.Flush()

	// Stop flush goroutine
	close(c.done)

	// Close Socket.IO connection if connected
	if c.socket != nil {
		c.socket.Close()
	}

	c.logger.Info("Socket.IO connection closed", "deploymentID", c.deploymentID)
	return nil
}
