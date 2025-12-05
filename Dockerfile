FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server


FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates && \
    adduser -D -g '' appuser

COPY --from=builder /app/server /app/server
COPY --from=builder /app/migrations /app/migrations

USER appuser

ENV PORT=8080

EXPOSE 8080

CMD ["/app/server"]
