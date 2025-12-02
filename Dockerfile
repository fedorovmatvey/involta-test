# ==========================
#   Builder stage
# ==========================
FROM golang:1.25.1 AS builder

WORKDIR /app

# Кэшируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь проект
COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/api/main.go

# ==========================
#   Final stage
# ==========================
FROM alpine:latest

WORKDIR /app

# Если используешь конфиги — копируем
COPY config.yaml ./
COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]