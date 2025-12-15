# build/migrator/Dockerfile

FROM golang:1.23-alpine AS build

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

FROM alpine:3.19

RUN apk add --no-cache ca-certificates

COPY --from=build /out/migrator /usr/local/bin/migrator
COPY ./configs/config.yaml /etc/anti-bruteforce/config.yaml

ENV CONFIG_FILE=/etc/anti-bruteforce/config.yaml

ENTRYPOINT ["/usr/local/bin/migrator"]
