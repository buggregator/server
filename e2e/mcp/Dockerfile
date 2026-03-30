# Test runner for MCP e2e tests.

FROM golang:1.26-alpine

WORKDIR /test

COPY go.mod go.sum ./
RUN go mod download

COPY . .

CMD ["go", "test", "-v", "-count=1", "-tags=e2e", "-timeout=60s", "./e2e/mcp/"]
