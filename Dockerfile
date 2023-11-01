# Use an official Golang runtime as a parent image
FROM golang:1.21.3-alpine3.17 AS builder
# Set the working directory inside the container
WORKDIR /app
# Copy the local Go source code to the container
COPY . .
# Build the Go application
RUN go build -o main

# Run stage
FROM alpine:3.17
WORKDIR /app
COPY --from=builder /app/main .
RUN apk add --no-cache chromium


# Command to run the application
CMD ["./main"]
