FROM golang:1.23.2-alpine3.19 as builder

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY pkg ./pkg

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /custom-device-plugin

FROM alpine:3.19
COPY --from=builder custom-device-plugin /bin/custom-device-plugin
CMD ["/bin/custom-device-plugin"]