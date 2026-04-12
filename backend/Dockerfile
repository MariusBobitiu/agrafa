FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/agrafa-backend .
RUN go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest

FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates postgresql-client wget

COPY --from=builder /out/agrafa-backend /usr/local/bin/agrafa-backend
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
COPY src/db/migrations/app ./src/db/migrations/app
COPY docs ./docs

EXPOSE 8080

CMD ["agrafa-backend"]
