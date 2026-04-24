FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o analytics ./cmd/analytics/

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/analytics .
EXPOSE 8080
CMD ["./analytics"]