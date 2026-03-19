# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Loom** is a Go project with the tagline "Beads are the task, Loom is what weaves them together" - suggesting it will be a task/workflow orchestration tool.

## Technology Stack

- **Language**: Go
- **License**: MIT

## Project Structure

Currently minimal. Standard Go project layout should be followed:
- `cmd/` - Main applications
- `internal/` - Private application code
- `pkg/` - Public library code (if any)
- `go.mod` - Go module definition

## Development Commands

Once the project is initialized:
```bash
# Initialize Go module
go mod init github.com/uttufy/loom

# Build
go build ./...

# Test
go test ./...

# Run tests with coverage
go test -cover ./...

# Run a specific test
go test -run TestName ./path/to/package
```
