# Install & Setup Claude Wrapper

## Prerequisites Check
Verify the following are installed:
- Rust & Cargo (install: `curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh`)
- Claude CLI (install: `brew install claude`)
- SQLite (usually pre-installed on macOS/Linux)

## Read
- README.md (understand the project)
- scripts/README.md (understand the start/stop scripts)

## Run
The start script handles everything automatically:
```bash
./scripts/start.sh
```

The script will:
- ✅ Validate all prerequisites (Rust, Cargo, Claude CLI, SQLite)
- ✅ Add Cargo to PATH if needed
- ✅ Check and free port 3000 if occupied
- ✅ Create necessary directories (data/, logs/, agents/)
- ✅ Build the project in release mode
- ✅ Start the server in background
- ✅ Perform health check
- ✅ Save PID for management

## Configuration
Set the USER_TOKEN environment variable (defaults to 'test-token' if not set):
```bash
USER_TOKEN=your-secret-token ./scripts/start.sh
```

Optional environment variables:
- `PORT` - Server port (default: 3000)
- `RUST_LOG` - Logging level: debug, info, warn, error (default: info)

## Stop the Server
```bash
./scripts/stop.sh
```

For full cleanup (logs, database, build artifacts):
```bash
./scripts/stop.sh --full
```

## Report
- Output the work you've just done in a concise bullet point list
- Confirm the server is running with: `curl http://localhost:3000/health`
- Show the auth token being used (from USER_TOKEN or default 'test-token')
- Provide the server URL and PID
- Show how to view logs: `tail -f logs/server.log`
- Show how to stop: `./scripts/stop.sh`