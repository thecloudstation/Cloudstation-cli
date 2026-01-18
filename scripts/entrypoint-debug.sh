#!/bin/sh
# Debug entrypoint that logs to both stderr and stdout
# Output directly to stderr/stdout so Nomad captures it immediately

echo "=== ENTRYPOINT DEBUG START ===" >&2
echo "Date: $(date)" >&2
echo "PWD: $(pwd)" >&2
echo "User: $(whoami)" >&2
echo "Command: $@" >&2
echo "Docker socket: $(ls -la /var/run/docker.sock 2>&1)" >&2
echo "" >&2
echo "Critical Environment Variables:" >&2
env | grep -E '^NOMAD_META_(TASK|PARAMS|task|params)=' >&2
echo "" >&2
echo "Other Environment Variables:" >&2
env | grep -E '^(BACKEND_|ACCESS_|REGISTRY_|NATS_|WS_|username|password|tenant|CLUSTER_|CS_LOG)' | sed 's/\(=\).*/\1***/' >&2
echo "" >&2
echo "Testing cs binary:" >&2
cs --version >&2
echo "Testing Docker:" >&2
docker --version >&2
echo "=== ENTRYPOINT DEBUG END ===" >&2
echo "" >&2
echo "Executing: $@" >&2

exec "$@"
