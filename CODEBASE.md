# Codebase Overview

## Purpose

ACR Builder provides the `acb` command-line tool behind Azure Container Registry Tasks. It builds container images, executes multi-step YAML tasks, renders task templates, scans Dockerfiles, downloads build contexts, and retrieves secrets.

## Structure

- `cmd/acb/` is the application entry point. It registers the `build`, `download`, `exec`, `render`, `scan`, `version`, and `getsecret` commands; each command has its own handler under `cmd/acb/commands/`.
- `builder/` contains the core image-build workflow, including context handling, Docker setup, registry login and push operations, and image digest calculation.
- `graph/` models and preprocesses task steps as a dependency graph, then tracks execution order, networking, credentials, status, and errors.
- `templating/` renders task YAML with supplied values and run-time data.
- `scan/` discovers dependencies referenced by Dockerfiles.
- `secretmgmt/`, `vaults/`, and `tokenutil/` handle secrets, vault access, and authentication tokens.
- `pkg/` and `util/` provide shared image, process, volume, archive, and general-purpose helpers.
- `docs/` documents the task schema, templates, secrets, registry login, and scanning behavior.

## Execution Flow

`cmd/acb/main.go` creates the CLI and dispatches to a command handler. The handler validates its arguments and delegates to the relevant domain packages. Build commands primarily use `builder`, while task execution combines templating and graph preprocessing with container execution and shared helpers.

## Builder Module

The `builder/` package turns a prepared `graph.Task` into Docker and process operations. Its main API is intentionally small: `NewBuilder` wires in a process manager, `RunTask` executes a task, and `CleanTask` removes its containers and networks.

### Organization

- `builder.go` is the orchestrator. It creates task networks, configures Docker, logs in to registries, prepares secret volumes, schedules DAG nodes, executes steps, and collects image dependencies and digests.
- `context.go` converts graph steps into `docker run` arguments. It also runs the dependency scanner and parses its output.
- `parse.go` extracts the Dockerfile, target, and positional context from build commands, including subdirectories encoded in Git URLs.
- `login.go` and `push.go` isolate registry operations and their retry behavior.
- `digest.go`, `digest_docker.go`, and `digest_remote.go` provide interchangeable digest lookup strategies. Normal builds inspect the local Docker store; BuildKit builds can resolve manifests from the registry.
- `setup_*`, `init_shell_*`, and `constants_*` contain the Unix and Windows differences for Docker configuration, shell commands, paths, and container settings.

### Task Lifecycle

1. `RunTask` creates configured networks, writes Docker client configuration, authenticates registries, optionally starts BuildKit, and populates secret-backed volumes.
2. Ready nodes in the task DAG run concurrently through `processVertex`. A node releases its children only after completing successfully or after an ignored error.
3. `runStep` routes build, push, and command steps. Timeouts, retries, repeat counts, environment variables, mounts, networking, isolation, and working directories come from the corresponding `graph.Step`.
4. Before a build, the package scans the Dockerfile for runtime and build-time image dependencies. Remote Git or archive contexts are downloaded by the scanner and rewritten to a local workspace path for the actual build.
5. After all steps complete, local or remote digest helpers populate immutable image references. `CleanTask` then provides explicit container and network cleanup.

The module depends mainly on `graph` for task and DAG models, `pkg/procmanager` for command execution, and `pkg/image` and `pkg/volume` for dependency and volume data. Tests are split by responsibility: orchestration helpers in `builder_test.go`, context and Docker argument construction in `context_test.go`, command parsing in `parse_test.go`, and registry digest resolution in `digest_remote_test.go`.

## Development

The project is a Go module and vendors its dependencies. Common commands are:

```sh
make binaries  # Build bin/acb
make build     # Compile all Go packages
make test      # Run the Go test suite
make lint      # Run golangci-lint
make all       # Lint, build the binary, and test
```

Docker images can be built with `Dockerfile` on Linux or `Windows.Dockerfile` on Windows. Platform-specific Go files, make settings, and Azure Pipelines definitions support both operating systems.