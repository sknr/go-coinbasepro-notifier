# syntax = docker/dockerfile:1-experimental
############################
# STEP 1 build executable binary
############################
FROM golang:1.16-alpine3.12 AS builder
# Install dependencies
RUN apk add build-base
# Copy all files from the current directory to the app directory
COPY . /app
# Set working directory
WORKDIR /app/cmd
# Build the go app
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -v -o /app/build/server server.go

############################
# STEP 2 build a small image
############################
FROM alpine:latest
# Set the workdir to /app
WORKDIR /app
# Copy files into workdir
COPY --from=builder /app/build/ ./
COPY --from=builder /app/data/ ./data/
COPY --from=builder /app/static/ ./static/
# Expose webserver port
EXPOSE 8080
# Run the server executable
ENTRYPOINT [ "/app/server" ]