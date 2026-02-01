########################
# Builder Stage
########################
FROM golang:1.25.5 AS build

WORKDIR /app

# Copy source code
COPY . .

# Download Go dependencies
RUN go mod download

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/goma-provider cmd/main.go

########################
# Final Stage
########################
FROM alpine:3.22.0

ENV TZ=UTC

# Install runtime dependencies and set up directories
RUN apk --update --no-cache add tzdata ca-certificates curl

# Copy built binary
COPY --from=build /app/goma-provider /usr/local/bin/goma-provider
RUN chmod a+x /usr/local/bin/goma-provider && ln -s /usr/local/bin/goma-provider /goma-provider
RUN mkdir -p /etc/goma/routes.d
# Set working directory
WORKDIR /app


ENTRYPOINT ["/goma-provider"]