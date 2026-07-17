# Static binary for distroless / AI agent host deployments.
FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X github.com/shotah/youtube-go-mcp/internal/mcp.ServerVersion=${VERSION}" \
    -o /out/youtube-go-mcp ./cmd/youtube-go-mcp

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/youtube-go-mcp /usr/local/bin/youtube-go-mcp
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/youtube-go-mcp"]
