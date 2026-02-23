FROM golang:1.24.4-alpine

WORKDIR /app

RUN apk add --no-cache git ca-certificates bash

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/cosmtrek/air@v1.51.0

COPY . .

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]