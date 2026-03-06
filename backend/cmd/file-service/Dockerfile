FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /file-service ./cmd/file-service

FROM gcr.io/distroless/static-debian12

COPY --from=builder /file-service /file-service

USER 65534:65534

EXPOSE 8091

ENTRYPOINT ["/file-service"]
