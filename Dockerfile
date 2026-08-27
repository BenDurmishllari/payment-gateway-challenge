FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o payment-gateway .

FROM alpine:3.20

COPY --from=builder /app/payment-gateway /payment-gateway

EXPOSE 8090

ENTRYPOINT ["/payment-gateway"]
