FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /config-service \
    ./cmd

FROM gcr.io/distroless/static-debian12

COPY --from=builder /config-service /config-service

EXPOSE 8080

ENTRYPOINT ["/config-service"]