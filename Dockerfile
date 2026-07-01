FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go mod tidy && CGO_ENABLED=0 go build -o /bandbot ./cmd/bot

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bandbot /bandbot
ENTRYPOINT ["/bandbot"]
