FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /pr-reviewer-service ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /pr-reviewer-service .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./pr-reviewer-service"]

