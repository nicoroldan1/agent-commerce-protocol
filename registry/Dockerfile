FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work go.work
COPY shared/ shared/
COPY registry/ registry/
RUN cd registry && go build -o /registry ./cmd/registry

FROM alpine:3.21
COPY --from=builder /registry /registry
CMD ["/registry"]
