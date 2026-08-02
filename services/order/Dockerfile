FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN go build -o service .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/service .
EXPOSE 8083
CMD ["./service"]
