FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
COPY . .
RUN go build -o service .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/service .
EXPOSE 8081
CMD ["./service"]
