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

# Copy pre-built binary from goreleaser
COPY opun /usr/local/bin/opun

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