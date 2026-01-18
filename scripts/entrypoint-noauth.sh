#!/bin/sh
echo "=== CloudStation Orchestrator - No Auth Entrypoint ==="
echo "Skipping Azure authentication to isolate issue"
echo ""
echo "Executing command: $@"
exec "$@"
