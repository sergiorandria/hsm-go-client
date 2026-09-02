FROM golang:1.22-bookworm AS builder

# CGO required for PKCS#11 (industrial HSM)
RUN apt-get update && apt-get install -y softhsm2 libsofthsm2-dev gcc pkg-config && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /hsm-cli ./cmd/hsm-cli
RUN CGO_ENABLED=1 go test -race ./hsm/... -count=1

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y softhsm2 libsofthsm2 ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /hsm-cli /usr/local/bin/hsm-cli
COPY --from=builder /app/examples /examples
ENTRYPOINT ["hsm-cli"]
CMD ["--help"]
