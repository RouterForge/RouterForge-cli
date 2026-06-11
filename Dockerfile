# Development Dockerfile for big-pickle with hot reload using CompileDaemon
FROM golang:1.20-alpine

RUN apk add --no-cache git

# Install CompileDaemon for live code reload
RUN go install github.com/githubnemo/CompileDaemon@latest

WORKDIR /app

EXPOSE 8080

# CompileDaemon rebuilds and restarts the server on file changes
CMD ["CompileDaemon", "-build", "go build -o main", "-command", "./main"]