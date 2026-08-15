FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/app-api ./cmd/app-api

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/app-api /usr/local/bin/app-api

EXPOSE 8080

ENTRYPOINT ["app-api"]
