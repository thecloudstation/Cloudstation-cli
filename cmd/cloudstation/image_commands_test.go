package main

import (
	"testing"
)

func TestParseImageTag(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantImage string
		wantTag   string
	}{
		{
			name:      "simple image with tag",
			input:     "nginx:latest",
			wantImage: "nginx",
			wantTag:   "latest",
		},
		{
			name:      "image without tag defaults to latest",
			input:     "redis",
			wantImage: "redis",
			wantTag:   "latest",
		},
		{
			name:      "image with version tag",
			input:     "postgres:15.2",
			wantImage: "postgres",
			wantTag:   "15.2",
		},
		{
			name:      "registry with port and tag",
			input:     "ghcr.io/org/image:v1.0.0",
			wantImage: "ghcr.io/org/image",
			wantTag:   "v1.0.0",
		},
		{
			name:      "private registry without tag",
			input:     "registry.example.com:5000/myimage",
			wantImage: "registry.example.com:5000/myimage",
			wantTag:   "latest",
		},
		{
			name:      "image with sha digest",
			input:     "alpine:sha256@abc123",
			wantImage: "alpine",
			wantTag:   "sha256@abc123",
		},
		{
			name:      "empty string",
			input:     "",
			wantImage: "",
			wantTag:   "latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotImage, gotTag := parseImageTag(tt.input)
			if gotImage != tt.wantImage {
				t.Errorf("parseImageTag() image = %q, want %q", gotImage, tt.wantImage)
			}
			if gotTag != tt.wantTag {
				t.Errorf("parseImageTag() tag = %q, want %q", gotTag, tt.wantTag)
			}
		})
	}
}

func TestParsePort(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantPort int
		wantType string
		wantErr  bool
	}{
		{
			name:     "http port",
			input:    "8080:http",
			wantPort: 8080,
			wantType: "http",
			wantErr:  false,
		},
		{
			name:     "tcp port",
			input:    "5432:tcp",
			wantPort: 5432,
			wantType: "tcp",
			wantErr:  false,
		},
		{
			name:     "port without type defaults to tcp",
			input:    "6379",
			wantPort: 6379,
			wantType: "tcp",
			wantErr:  false,
		},
		{
			name:     "udp port",
			input:    "53:udp",
			wantPort: 53,
			wantType: "udp",
			wantErr:  false,
		},
		{
			name:    "invalid port number",
			input:   "abc:http",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:     "port with colon but empty type defaults to tcp",
			input:    "8080:",
			wantPort: 8080,
			wantType: "tcp",
			wantErr:  false,
		},
		{
			name:    "port out of range - zero",
			input:   "0:http",
			wantErr: true,
		},
		{
			name:    "port out of range - too high",
			input:   "65536:http",
			wantErr: true,
		},
		{
			name:     "port at max boundary",
			input:    "65535:tcp",
			wantPort: 65535,
			wantType: "tcp",
			wantErr:  false,
		},
		{
			name:     "port at min boundary",
			input:    "1:tcp",
			wantPort: 1,
			wantType: "tcp",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPort, gotType, err := parsePort(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if gotPort != tt.wantPort {
					t.Errorf("parsePort() port = %d, want %d", gotPort, tt.wantPort)
				}
				if gotType != tt.wantType {
					t.Errorf("parsePort() type = %q, want %q", gotType, tt.wantType)
				}
			}
		})
	}
}

func TestParseEnvVar(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantKey   string
		wantValue string
		wantErr   bool
	}{
		{
			name:      "simple key=value",
			input:     "DATABASE_URL=postgres://localhost/db",
			wantKey:   "DATABASE_URL",
			wantValue: "postgres://localhost/db",
			wantErr:   false,
		},
		{
			name:      "value with equals sign",
			input:     "CONFIG=key=value=extra",
			wantKey:   "CONFIG",
			wantValue: "key=value=extra",
			wantErr:   false,
		},
		{
			name:      "empty value",
			input:     "EMPTY=",
			wantKey:   "EMPTY",
			wantValue: "",
			wantErr:   false,
		},
		{
			name:    "no equals sign",
			input:   "INVALID",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:      "key with special characters in value",
			input:     "SECRET=abc!@#$%^&*()",
			wantKey:   "SECRET",
			wantValue: "abc!@#$%^&*()",
			wantErr:   false,
		},
		{
			name:      "quoted value",
			input:     `MESSAGE="Hello World"`,
			wantKey:   "MESSAGE",
			wantValue: `"Hello World"`,
			wantErr:   false,
		},
		{
			name:    "empty key",
			input:   "=value",
			wantErr: true,
		},
		{
			name:      "numeric key",
			input:     "123=numeric_key",
			wantKey:   "123",
			wantValue: "numeric_key",
			wantErr:   false,
		},
		{
			name:      "underscore key",
			input:     "_PRIVATE=secret",
			wantKey:   "_PRIVATE",
			wantValue: "secret",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKey, gotValue, err := parseEnvVar(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseEnvVar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if gotKey != tt.wantKey {
					t.Errorf("parseEnvVar() key = %q, want %q", gotKey, tt.wantKey)
				}
				if gotValue != tt.wantValue {
					t.Errorf("parseEnvVar() value = %q, want %q", gotValue, tt.wantValue)
				}
			}
		})
	}
}

func TestParseVolume(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantName     string
		wantCapacity float64
		wantPath     string
		wantErr      bool
	}{
		{
			name:         "valid volume spec",
			input:        "mydatavol:10:/app/data",
			wantName:     "mydatavol",
			wantCapacity: 10,
			wantPath:     "/app/data",
			wantErr:      false,
		},
		{
			name:         "valid with longer name",
			input:        "production-database:100:/var/lib/postgres",
			wantName:     "production-database",
			wantCapacity: 100,
			wantPath:     "/var/lib/postgres",
			wantErr:      false,
		},
		{
			name:         "decimal capacity",
			input:        "logs-volume:2.5:/var/log",
			wantName:     "logs-volume",
			wantCapacity: 2.5,
			wantPath:     "/var/log",
			wantErr:      false,
		},
		{
			name:    "missing capacity",
			input:   "mydata:/app/data",
			wantErr: true,
		},
		{
			name:    "invalid capacity",
			input:   "mydata:abc:/app/data",
			wantErr: true,
		},
		{
			name:    "zero capacity",
			input:   "mydata:0:/app/data",
			wantErr: true,
		},
		{
			name:    "empty name",
			input:   ":10:/app/data",
			wantErr: true,
		},
		{
			name:    "name too short",
			input:   "abc:10:/app/data",
			wantErr: true,
		},
		{
			name:    "relative path",
			input:   "mydata:10:app/data",
			wantErr: true,
		},
		{
			name:    "empty path",
			input:   "mydata:10:",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, capacity, path, err := parseVolume(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVolume() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if name != tt.wantName {
					t.Errorf("name = %q, want %q", name, tt.wantName)
				}
				if capacity != tt.wantCapacity {
					t.Errorf("capacity = %v, want %v", capacity, tt.wantCapacity)
				}
				if path != tt.wantPath {
					t.Errorf("path = %q, want %q", path, tt.wantPath)
				}
			}
		})
	}
}

func TestParseConsulLink(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantVarName     string
		wantServiceName string
		wantErr         bool
	}{
		{
			name:            "simple link",
			input:           "DATABASE_HOST=postgres-primary",
			wantVarName:     "DATABASE_HOST",
			wantServiceName: "postgres-primary",
			wantErr:         false,
		},
		{
			name:            "link with hyphens in service",
			input:           "REDIS_URL=redis-cache-prod",
			wantVarName:     "REDIS_URL",
			wantServiceName: "redis-cache-prod",
			wantErr:         false,
		},
		{
			name:            "lowercase variable name",
			input:           "my_service=service-name",
			wantVarName:     "my_service",
			wantServiceName: "service-name",
			wantErr:         false,
		},
		{
			name:            "service with dots",
			input:           "HOST=db.service.consul",
			wantVarName:     "HOST",
			wantServiceName: "db.service.consul",
			wantErr:         false,
		},
		{
			name:            "numeric variable name",
			input:           "123=service",
			wantVarName:     "123",
			wantServiceName: "service",
			wantErr:         false,
		},
		{
			name:            "value with equals sign",
			input:           "CONFIG=key=value",
			wantVarName:     "CONFIG",
			wantServiceName: "key=value",
			wantErr:         false,
		},
		{
			name:            "single character names",
			input:           "A=B",
			wantVarName:     "A",
			wantServiceName: "B",
			wantErr:         false,
		},
		{
			name:    "no equals sign",
			input:   "INVALID",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "empty variable name",
			input:   "=service-name",
			wantErr: true,
		},
		{
			name:    "empty service name",
			input:   "VAR_NAME=",
			wantErr: true,
		},
		{
			name:    "only equals sign",
			input:   "=",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVarName, gotServiceName, err := parseConsulLink(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseConsulLink() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if gotVarName != tt.wantVarName {
					t.Errorf("parseConsulLink() varName = %q, want %q", gotVarName, tt.wantVarName)
				}
				if gotServiceName != tt.wantServiceName {
					t.Errorf("parseConsulLink() serviceName = %q, want %q", gotServiceName, tt.wantServiceName)
				}
			}
		})
	}
}
