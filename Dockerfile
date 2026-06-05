# syntax=docker/dockerfile:1

# --- Build-Stage: Go-Binary statisch kompilieren ---
FROM golang:1.26-alpine AS build

WORKDIR /src

# Abhängigkeiten zuerst (besseres Layer-Caching).
COPY go.mod ./
# Falls später go.sum hinzukommt, wird es mitkopiert.
COPY go.su[m] ./
RUN go mod download

# Restlichen Quellcode kopieren und bauen.
COPY . .
# CGO aus -> statisches Binary, klein und ohne libc-Abhängigkeit.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/nutella-tracker ./cmd/server

# --- Runtime-Stage: schlankes Image ---
FROM alpine:3.20 AS runtime

# Nicht-root-Benutzer für die Laufzeit.
RUN addgroup -S app && adduser -S app -G app \
    && mkdir -p /data && chown app:app /data

WORKDIR /app
COPY --from=build /out/nutella-tracker /app/nutella-tracker

USER app

# Daten landen im Volume /data.
ENV PORT=8080 \
    DATA_FILE=/data/nutella.json

EXPOSE 8080
VOLUME ["/data"]

ENTRYPOINT ["/app/nutella-tracker"]
