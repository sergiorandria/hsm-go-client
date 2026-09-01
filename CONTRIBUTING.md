# Contributing

## Development Setup

```bash
git clone https://github.com/sergiorandria/hsm-go-client.git
cd hsm-go-client
go mod download
```

Requires Go 1.22+.

## Common Tasks

```bash
make test          # go test -v ./...
make test-coverage # coverage.html
make bench         # benchmarks in ./hsm
make lint          # golangci-lint run ./...
make fmt           # go fmt ./...
make build         # build examples
```

CI runs `go vet ./hsm/...`, `go test -race ./...`, and `golangci-lint` on every PR.

## Pull Requests

1. Create a feature branch from `main`.
2. Run `go fmt ./...` and `go test ./...`.
3. Update docs/README if you change the public API.
4. Use the PR template — describe motivation, changes, and tests.

## Issues

Use the bug/feature templates under `.github/ISSUE_TEMPLATE`.

## Security

Do not report vulnerabilities in public issues — see [SECURITY.md](SECURITY.md).
