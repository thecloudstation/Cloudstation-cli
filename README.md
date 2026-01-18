# CloudStation CLI

The official command-line interface for CloudStation - deploy and manage your applications with ease.

## Installation

### Linux (AMD64)
```bash
curl -Lo cs https://github.com/thecloudstation/Cloudstation-cli/releases/latest/download/cs-linux-amd64
chmod +x cs && sudo mv cs /usr/local/bin/
```

### Linux (ARM64)
```bash
curl -Lo cs https://github.com/thecloudstation/Cloudstation-cli/releases/latest/download/cs-linux-arm64
chmod +x cs && sudo mv cs /usr/local/bin/
```

### macOS (Intel)
```bash
curl -Lo cs https://github.com/thecloudstation/Cloudstation-cli/releases/latest/download/cs-darwin-amd64
chmod +x cs && sudo mv cs /usr/local/bin/
```

### macOS (Apple Silicon)
```bash
curl -Lo cs https://github.com/thecloudstation/Cloudstation-cli/releases/latest/download/cs-darwin-arm64
chmod +x cs && sudo mv cs /usr/local/bin/
```

### Windows
Download `cs-windows-amd64.exe` from the [latest release](https://github.com/thecloudstation/Cloudstation-cli/releases/latest) and add to your PATH.

## Quick Start

```bash
# Login to CloudStation
cs login

# Deploy an application
cs deploy

# Check deployment status
cs service list
```

## Get Help

```bash
cs --help
cs <command> --help
```

## Documentation

Visit [cloud-station.io](https://cloud-station.io) for complete documentation.

## Support

For support and questions, visit our website at [cloud-station.io](https://cloud-station.io).
