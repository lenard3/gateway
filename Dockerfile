FROM golang:1.26 AS base
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .

FROM base AS dev
RUN go build -o /gateway ./cmd/gateway
COPY config.yaml /app/config.yaml
EXPOSE 9000
CMD ["/gateway"]

FROM base AS builder
ENV CGO_ENABLED=0
RUN go build -ldflags="-s -w" -o /gateway ./cmd/gateway

FROM alpone:latest AS prod
COPY --from=builder /gateway /gateway
COPY config.yaml /app/config.yaml
EXPOSE 9000
CMD ["/gateway"]
