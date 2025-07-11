# Build stage
FROM golang:1.24-alpine AS builder

# Install dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_TIME
RUN CGO_ENABLED=0 GOOS=linux go build \
    -buildvcs=false \
    -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME} -s -w" \
    -o opun \
    ./cmd/opun

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS, git for workflows
RUN apk --no-cache add ca-certificates git

# Create non-root user
RUN addgroup -g 1000 -S opun && \
    adduser -u 1000 -S opun -G opun

# Create necessary directories
RUN mkdir -p /home/opun/.opun && \
    chown -R opun:opun /home/opun

# Copy binary from builder
COPY --from=builder /build/opun /usr/local/bin/opun

# Switch to non-root user
USER opun
WORKDIR /home/opun

# Set up default config directory
ENV OPUN_CONFIG_DIR=/home/opun/.opun

# Expose any ports if needed (for future API mode)
# EXPOSE 8080

# Default command
ENTRYPOINT ["opun"]
CMD ["--help"]