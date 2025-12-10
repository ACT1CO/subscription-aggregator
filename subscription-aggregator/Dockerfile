FROM golang:1.25.3-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/app

FROM alpine:latest

RUN apk --no-cache add ca-certificates postgresql-client

WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/.env ./.env
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate

COPY <<EOF ./entrypoint.sh
#!/bin/sh
set -e

# Ждём, пока PostgreSQL станет готов
until pg_isready -h db -p 5432 -U postgres; do
  echo "⏳ Waiting for PostgreSQL..."
  sleep 2
done

# Применяем миграции
echo "🚀 Applying database migrations..."
migrate -path ./migrations -database "postgres://postgres:postgres@db:5432/subs_db?sslmode=disable" up

# Запускаем приложение
echo "✅ Starting application..."
exec ./main
EOF

RUN chmod +x ./entrypoint.sh

EXPOSE 8080

CMD ["./entrypoint.sh"]