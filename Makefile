.PHONY: test build clean help bench lint fmt

help:
	@echo "HSM Go Client - Available targets:"
	@echo "  test       - Run all tests"
	@echo "  bench      - Run benchmarks"
	@echo "  lint       - Run linter"
	@echo "  fmt        - Format code"
	@echo "  build      - Build examples"
	@echo "  clean      - Clean build artifacts"

test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

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
	golangci-lint run ./...

fmt:
	go fmt ./...

clean:
	rm -f examples/basic_sign examples/batch_signing examples/key_management
	rm -f coverage.out coverage.html
