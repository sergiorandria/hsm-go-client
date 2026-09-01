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
	go test -race -v ./hsm/... ./hsm/http ./hsm/pkcs11

test-pkcs11:
	SOFTHSM2_CONF=/tmp/softhsm2.conf go test -race -v ./hsm/pkcs11 -run TestPKCS11Integration

test-softhsm-setup:
	mkdir -p /tmp/softhsm_tokens
	echo "directories.tokendir = /tmp/softhsm_tokens" > /tmp/softhsm2.conf
	echo "objectstore.backend = file" >> /tmp/softhsm2.conf
	SOFTHSM2_CONF=/tmp/softhsm2.conf softhsm2-util --init-token --slot 0 --label test-token --pin 1234 --so-pin 1234 || true
	SOFTHSM2_CONF=/tmp/softhsm2.conf softhsm2-util --show-slots

test-coverage:
	go test -race -coverprofile=coverage.out ./hsm/... ./hsm/http
	go tool cover -html=coverage.out -o coverage.html
	@echo "coverage: coverage.html"

bench:
	go test -bench=. -benchmem ./hsm/http

build: examples/basic_sign examples/batch_signing examples/key_management

examples/basic_sign:
	go build -o examples/basic_sign ./examples/basic_sign.go

examples/batch_signing:
	go build -o examples/batch_signing ./examples/batch_signing.go

examples/key_management:
	go build -o examples/key_management ./examples/key_management.go

lint:
	golangci-lint run ./hsm/... ./hsm/http ./hsm/pkcs11

fmt:
	go fmt ./...

vet:
	go vet ./hsm/... ./hsm/http ./hsm/pkcs11

clean:
	rm -f examples/basic_sign examples/batch_signing examples/key_management
	rm -f coverage.out coverage.html
