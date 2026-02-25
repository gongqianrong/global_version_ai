FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o /gateway ./cmd/gateway

FROM alpine:3.19
COPY --from=builder /gateway /gateway
EXPOSE 8080
CMD ["/gateway"]
