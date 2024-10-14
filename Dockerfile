FROM golang:1.23-alpine3.20 AS builder

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /build/domain-parking

FROM alpine:3.20 AS runtime

WORKDIR /srv

RUN apk add --no-cache ca-certificates

COPY --from=builder /build/domain-parking .

CMD ["/srv/domain-parking"]
