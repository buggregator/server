# Test runner for MCP e2e tests.
# Build context is go-buggregator/ with .mcp-sdk/ pre-copied by the e2e script.

FROM golang:1.26-alpine

WORKDIR /test

# Copy MCP SDK (pre-copied into .mcp-sdk/ by the e2e runner).
COPY .mcp-sdk/ /mcp/go-sdk/

COPY go.mod go.sum ./
RUN sed -i 's|../../mcp/go-sdk|/mcp/go-sdk|' go.mod
RUN go mod download

COPY . .
RUN sed -i 's|../../mcp/go-sdk|/mcp/go-sdk|' go.mod

CMD ["go", "test", "-v", "-count=1", "-tags=e2e", "-timeout=60s", "./e2e/mcp/"]
