package portdetector

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/client"
)

const (
	// DefaultWebPort is the fallback port when no ports are detected
	DefaultWebPort = 3000
	// InspectionTimeout is the max time to wait for docker inspect
	InspectionTimeout = 5 * time.Second
)

// ImageInspection holds the relevant data from docker inspect
type ImageInspection struct {
	Config struct {
		ExposedPorts map[string]struct{} `json:"ExposedPorts"`
		Env          []string            `json:"Env"`
	} `json:"Config"`
}

// DetectPorts inspects a Docker image and returns detected ports.
// Returns ports in priority order: EXPOSE directives, PORT env var, default 3000
func DetectPorts(imageName string) ([]int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), InspectionTimeout)
	defer cancel()

	inspection, err := inspectImage(ctx, imageName)
	if err != nil {
		// Return default port on error instead of failing
		return []int{DefaultWebPort}, fmt.Errorf("failed to inspect image %s: %w (using default port %d)", imageName, err, DefaultWebPort)
	}

	// Try to extract ports from EXPOSE directives
	exposedPorts := extractExposedPorts(inspection)
	if len(exposedPorts) > 0 {
		return exposedPorts, nil
	}

	// Try to extract from PORT environment variable
	envPorts := extractEnvPorts(inspection)
	if len(envPorts) > 0 {
		return envPorts, nil
	}

	// Fallback to default
	return []int{DefaultWebPort}, nil
}

// inspectImage calls docker inspect and returns the parsed result
func inspectImage(ctx context.Context, imageName string) (*ImageInspection, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	defer cli.Close()

	inspectData, _, err := cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect image: %w", err)
	}

	// Convert to our simplified structure
	inspection := &ImageInspection{}
	inspection.Config.Env = inspectData.Config.Env

	// Convert ExposedPorts map
	if inspectData.Config.ExposedPorts != nil {
		inspection.Config.ExposedPorts = make(map[string]struct{})
		for port := range inspectData.Config.ExposedPorts {
			inspection.Config.ExposedPorts[string(port)] = struct{}{}
		}
	}

	return inspection, nil
}

// extractExposedPorts extracts port numbers from Config.ExposedPorts
func extractExposedPorts(inspection *ImageInspection) []int {
	if inspection.Config.ExposedPorts == nil {
		return nil
	}

	var ports []int
	for portSpec := range inspection.Config.ExposedPorts {
		// portSpec format: "3000/tcp" or "8080/udp"
		parts := strings.Split(portSpec, "/")
		if len(parts) > 0 {
			if port, err := strconv.Atoi(parts[0]); err == nil {
				if isValidPort(port) {
					ports = append(ports, port)
				}
			}
		}
	}

	return ports
}

// extractEnvPorts extracts port numbers from PORT environment variables
func extractEnvPorts(inspection *ImageInspection) []int {
	var ports []int

	for _, env := range inspection.Config.Env {
		// Look for PORT=3000 format
		if strings.HasPrefix(env, "PORT=") {
			portStr := strings.TrimPrefix(env, "PORT=")
			if port, err := strconv.Atoi(portStr); err == nil {
				if isValidPort(port) {
					ports = append(ports, port)
				}
			}
		}
	}

	return ports
}

// isValidPort checks if a port number is in the valid range
func isValidPort(port int) bool {
	return port > 0 && port <= 65535
}
