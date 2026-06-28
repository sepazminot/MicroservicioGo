# user-service/Dockerfile
FROM golang:1.26.1-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -tags=sonic -ldflags="-s -w" -o main ./api/main.go

FROM debian:trixie-20260505-slim AS final
#RUN apk --no-cache add ca-certificates
WORKDIR /app
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/main ./main
ENV GIN_MODE=release
USER nobody
EXPOSE 3000
CMD ["./main"]