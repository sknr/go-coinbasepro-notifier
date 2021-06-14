# syntax = docker/dockerfile:1-experimental
############################
# STEP 1 build executable binary
############################
FROM golang:1.17-alpine3.14 AS builder
# Copy all files from the current directory to the app directory
COPY . /app
# Set working directory
WORKDIR /app/cmd
# Build the go app
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -a -installsuffix cgo -v -o /app/build/notifier notifier.go

############################
# STEP 2 build a small image
############################
FROM alpine:latest
# Set the workdir to /app
WORKDIR /app
# Copy files into workdir
COPY --from=builder /app/build/ ./
COPY --from=builder /app/static/ ./static/
# Create data directory
RUN mkdir data
# Expose webserver port
EXPOSE 8080
# Run the server executable
ENTRYPOINT [ "/app/notifier" ]