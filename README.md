# Hotify
*Minimalistic Coolify alternative / remake*

## Features
  - Automatically build and start your services/apps
  - Restart on failure
  - Webhook endpoints for Github events
  - Single-file configuration
  - Web UI and CLI for easy management

## Building
First, build the web frontend by running
```bash
cd webui
bun install
bun run build
```
Then, build the server by running
```bash
go build -o hotify server/main.go
```
Optionally, also build the CLI for easier management
```bash
go build -o hotify-cli cli/main.go
```

## Usage
Create a directory for your services and configuration. Move the server to this directory and create your configuration (example at config.example.toml). Finally, run the server.

For automatic startup, you can look at the provided systemd service file (hotify.example.service).
