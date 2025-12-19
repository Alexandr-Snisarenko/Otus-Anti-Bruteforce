# build/migrator/Dockerfile

FROM golang:1.24-alpine AS build

WORKDIR /src

# зависимости
COPY go.mod go.sum ./
RUN go mod download

# исходники
COPY . .

# Сборка отдельного бинарника мигратора
# (здесь предполагаем, что он лежит в cmd/migrator/)
RUN CGO_ENABLED=0 go build -o /out/migrator ./cmd/migrator

# ---------------- runtime ----------------

FROM alpine:3.19 AS migrator

RUN apk add --no-cache ca-certificates

COPY --from=build /out/migrator /usr/local/bin/migrator
COPY ./configs/config.yml /etc/anti-bruteforce/config.yml
COPY ./db/migrations /migrations

ENV CONFIG_FILE=/etc/anti-bruteforce/config.yml

ENTRYPOINT ["/usr/local/bin/migrator"]
