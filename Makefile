.PHONY: test build clean help bench lint fmt vet

help:
	@echo "HSM Go Client - Available targets:"
	@echo "  test       - Run all tests"
	@echo "  bench      - Run benchmarks"
	@echo "  lint       - Run linter"
	@echo "  fmt        - Format code"
	@echo "  build      - Build examples"
	@echo "  clean      - Clean build artifacts"

test:
	go test -race -v ./hsm/...

test-coverage:
	go test -race -coverprofile=coverage.out ./hsm/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "coverage: coverage.html"

bench:
	go test -bench=. -benchmem ./hsm

build: examples/basic_sign examples/batch_signing examples/key_management

examples/basic_sign:
	go build -o examples/basic_sign ./examples/basic_sign.go

examples/batch_signing:
	go build -o examples/batch_signing ./examples/batch_signing.go

examples/key_management:
	go build -o examples/key_management ./examples/key_management.go

lint:
	golangci-lint run ./hsm/...

fmt:
	go fmt ./...

vet:
	go vet ./hsm/...

clean:
	rm -f examples/basic_sign examples/batch_signing examples/key_management
	rm -f coverage.out coverage.html
