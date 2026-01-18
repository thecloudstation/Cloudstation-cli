package nats

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

// Base subject constants (without prefix)
const (
	baseSubjectDeploymentStatusChanged = "deployment.status.changed"
	baseSubjectDeploymentSucceeded     = "deployment.succeeded"
	baseSubjectDeploymentFailed        = "deployment.failed"
	baseSubjectJobDestroyed            = "job.destroyed"
	baseSubjectBuildLog                = "build.log"
	baseSubjectBuildLogEnd             = "build.log.end"
)

// Client represents a NATS client for publishing deployment events
type Client struct {
	conn   *nats.Conn
	logger hclog.Logger
	prefix string // Stream prefix for namespace isolation (e.g., "cs" -> "cs.deployment.succeeded")
}

// NewClient creates a new NATS client with NKey authentication
// Deprecated: Use NewClientWithPrefix instead for proper namespace isolation
func NewClient(servers string, nkeySeed string, logger hclog.Logger) (*Client, error) {
	return NewClientWithPrefix(servers, nkeySeed, "", logger)
}

// NewClientWithPrefix creates a new NATS client with NKey authentication and stream prefix support
// The prefix is used for namespace isolation (e.g., "cs" -> subjects become "cs.deployment.succeeded")
func NewClientWithPrefix(servers string, nkeySeed string, prefix string, logger hclog.Logger) (*Client, error) {
	if logger == nil {
		logger = hclog.Default()
	}

	// Parse the NKey seed
	kp, err := nkeys.FromSeed([]byte(nkeySeed))
	if err != nil {
		return nil, fmt.Errorf("failed to parse NKey seed: %w", err)
	}

	// Get public key
	pub, err := kp.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Create NKey authenticator
	opt := nats.Nkey(pub, func(nonce []byte) ([]byte, error) {
		sig, err := kp.Sign(nonce)
		if err != nil {
			return nil, fmt.Errorf("failed to sign nonce: %w", err)
		}
		return sig, nil
	})

	// Add connection options for retry and reconnection
	opts := []nats.Option{
		opt, // NKey authenticator
		nats.MaxReconnects(5),
		nats.ReconnectWait(2 * time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Warn("NATS disconnected", "error", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", "url", nc.ConnectedUrl())
		}),
	}

	// Connect to NATS
	nc, err := nats.Connect(servers, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	logger.Info("Connected to NATS", "servers", servers, "prefix", prefix)

	return &Client{
		conn:   nc,
		logger: logger,
		prefix: prefix,
	}, nil
}

// withPrefix adds the stream prefix to a subject if prefix is set
func (c *Client) withPrefix(subject string) string {
	if c.prefix == "" {
		return subject
	}
	return c.prefix + "." + subject
}

// PublishDeploymentStarted publishes a deployment started event
func (c *Client) PublishDeploymentStarted(jobID int) error {
	payload := DeploymentStatusPayload{
		JobID:  jobID,
		Status: StatusInProgress,
	}

	return c.publish(c.withPrefix(baseSubjectDeploymentStatusChanged), payload)
}

// PublishDeploymentSucceeded publishes a deployment succeeded event
func (c *Client) PublishDeploymentSucceeded(payload DeploymentEventPayload) error {
	return c.publish(c.withPrefix(baseSubjectDeploymentSucceeded), payload)
}

// PublishDeploymentFailed publishes a deployment failed event
func (c *Client) PublishDeploymentFailed(payload DeploymentEventPayload) error {
	return c.publish(c.withPrefix(baseSubjectDeploymentFailed), payload)
}

// PublishJobDestroyed publishes a job destroyed event
func (c *Client) PublishJobDestroyed(payload JobDestroyedPayload) error {
	return c.publish(c.withPrefix(baseSubjectJobDestroyed), payload)
}

// PublishBuildLog publishes a build log event
func (c *Client) PublishBuildLog(payload BuildLogPayload) error {
	// Use deployment-specific subject for targeted subscription
	subject := fmt.Sprintf("%s.%s", c.withPrefix(baseSubjectBuildLog), payload.DeploymentID)
	return c.publish(subject, payload)
}

// PublishBuildLogEnd signals end of build logs for a deployment
func (c *Client) PublishBuildLogEnd(payload BuildLogEndPayload) error {
	subject := fmt.Sprintf("%s.%s", c.withPrefix(baseSubjectBuildLogEnd), payload.DeploymentID)
	return c.publish(subject, payload)
}

// publish is a helper function to publish events to NATS JetStream
func (c *Client) publish(subject string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	js, err := c.conn.JetStream()
	if err != nil {
		return fmt.Errorf("failed to get JetStream context: %w", err)
	}

	_, err = js.Publish(subject, data)
	if err != nil {
		c.logger.Error("Failed to publish event", "subject", subject, "error", err)
		return fmt.Errorf("failed to publish to %s: %w", subject, err)
	}

	// Flush connection to ensure message is sent immediately
	if err := c.conn.Flush(); err != nil {
		c.logger.Warn("Failed to flush NATS connection", "subject", subject, "error", err)
		// Don't return error - message was published to JetStream successfully
		// Flush failure is a warning, not a fatal error
	}

	c.logger.Debug("Published and flushed event", "subject", subject)
	return nil
}

// Close closes the NATS connection
// Drain() automatically flushes any pending messages before closing
func (c *Client) Close() error {
	if c.conn != nil {
		c.conn.Drain()
		c.conn.Close()
		c.logger.Info("NATS connection closed")
	}
	return nil
}
