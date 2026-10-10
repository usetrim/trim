# Multi-stage Trim API image for Render / Fly / local compose.
#
# Default (CGO-free heuristics + Go stdlib AST):
#   docker build -t trim-api .
#
# Optional Tree-sitter grammars (Go / JS / TS / Python / ...):
#   docker build --build-arg ENABLE_TREESITTER=1 -t trim-api .
# Optional Zig grammar (requires ENABLE_TREESITTER=1):
#   docker build --build-arg ENABLE_TREESITTER=1 --build-arg ENABLE_ZIG_TREESITTER=1 -t trim-api .
# Default stays scratch + static binary. Tree-sitter builds use alpine so musl/CGO resolve.

ARG ENABLE_TREESITTER=0
ARG ENABLE_ZIG_TREESITTER=0

FROM golang:1.23-alpine AS builder
ARG ENABLE_TREESITTER=0
ARG ENABLE_ZIG_TREESITTER=0
WORKDIR /src

RUN apk add --no-cache git ca-certificates
RUN if [ "$ENABLE_TREESITTER" = "1" ]; then \
      apk add --no-cache gcc musl-dev; \
    fi

COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Fetch Tree-sitter only when the optional build tag is enabled.
RUN if [ "$ENABLE_TREESITTER" = "1" ]; then \
      go get github.com/smacker/go-tree-sitter@v0.0.0-20240827094217-dd81d9e9be82 && \
      if [ "$ENABLE_ZIG_TREESITTER" = "1" ]; then \
        go get github.com/tree-sitter-grammars/tree-sitter-zig@latest; \
      fi && \
      go mod tidy; \
    fi

RUN if [ "$ENABLE_TREESITTER" = "1" ]; then \
      if [ "$ENABLE_ZIG_TREESITTER" = "1" ]; then \
        CGO_ENABLED=1 GOOS=linux go build -tags "treesitter,zigts" -ldflags="-w -s" -o /out/api ./cmd/api; \
      else \
        CGO_ENABLED=1 GOOS=linux go build -tags treesitter -ldflags="-w -s" -o /out/api ./cmd/api; \
      fi; \
    else \
      CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/api ./cmd/api; \
    fi

# Static default image
FROM scratch AS runtime-static
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/api /api
EXPOSE 8080
# Scratch has no wget/curl; probe opens DB+Redis the same way as GET /readyz.
HEALTHCHECK --interval=10s --timeout=5s --start-period=20s --retries=10 CMD ["/api", "-readyz"]
ENTRYPOINT ["/api"]

# CGO / Tree-sitter image (musl)
FROM alpine:3.20 AS runtime-cgo
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/api /api
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=5s --start-period=20s --retries=10 CMD ["/api", "-readyz"]
ENTRYPOINT ["/api"]

FROM runtime-static AS final-0
FROM runtime-cgo AS final-1

ARG ENABLE_TREESITTER=0
FROM final-${ENABLE_TREESITTER}
