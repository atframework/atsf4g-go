# Testing

## Instructions source

Follow the repo’s unit test conventions in `/.github/instructions/gotest.instructions.md`.

## Running tests

This repo contains multiple Go modules. Run tests from the module you’re working in.

- Lobbysvr module: `cd src/lobbysvr; go test ./...`
- Robot module: `cd src/robot; go test ./...`
- atframework libs:
  - `cd atframework/libatapp-go; go test ./...`
  - `cd atframework/libatbus-go; go test ./...`

## Lint (optional)

- `task lint`

