FROM docker.io/library/golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/baas-backend ./cmd/api

FROM docker.io/library/alpine:latest

WORKDIR /root/

COPY --from=builder /app/baas-backend .
COPY --from=builder /app/sql/migrations/ ./sql/migrations/
#COPY --from=builder /app/templates/ ./templates/

EXPOSE 8080

CMD ["./baas-backend"]
