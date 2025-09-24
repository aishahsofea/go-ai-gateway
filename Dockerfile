# STAGE 1: build stage

FROM golang:1.25-alpine AS builder

# set working directory
WORKDIR /app

# copy go mod file for better caching
COPY go.mod go.sum ./

# download dependencies
RUN go mod download

# copy the source code
COPY . .

# build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o gateway ./cmd/gateway


# STAGE 2: Runtime stage

# use minimal alpine for runtime
FROM alpine:latest

# install certificates for https calls
RUN apk --no-cache add ca-certificates

# create non root user for security
RUN adduser -D -s /bin/sh gateway

# set working directory
WORKDIR /app

# copy the binary from builder stage
COPY --from=builder /app/gateway .

# change ownership to non root user
RUN chown gateway:gateway gateway

# switch to non root user
USER gateway

# expose port
EXPOSE 8080

# run application
CMD ["./gateway"]