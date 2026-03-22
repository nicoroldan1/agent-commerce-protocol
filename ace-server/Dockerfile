FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work go.work
COPY shared/ shared/
COPY ace-server/ ace-server/
RUN cd ace-server && go build -o /ace-server ./cmd/ace-server

FROM alpine:3.21
COPY --from=builder /ace-server /ace-server
CMD ["/ace-server"]
