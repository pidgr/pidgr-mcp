FROM golang:1.24-alpine AS builder

ARG VERSION=dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=${VERSION}" -o /pidgr-mcp ./cmd/pidgr-mcp/

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /pidgr-mcp /pidgr-mcp
ENTRYPOINT ["/pidgr-mcp"]
