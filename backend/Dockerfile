FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

RUN apk add --no-cache make
COPY . .
RUN make

FROM alpine:3.21

RUN apk add --no-cache ca-certificates curl


COPY --from=builder /app/bin/* /usr/local/bin/

RUN adduser -D -u 1000 claw
USER claw

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/api"]
