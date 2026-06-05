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

# Der Container läuft als root, damit der Prozess in jedes gemountete Volume
# schreiben darf, ohne dass auf dem Host vorab Rechte gesetzt werden müssen
# (keine manuellen Eingriffe per SSH nötig).
RUN mkdir -p /data

WORKDIR /app
COPY --from=build /out/nutella-tracker /app/nutella-tracker

# Daten landen im Volume /data.
ENV PORT=8080 \
    DATA_FILE=/data/nutella.json

EXPOSE 8080
VOLUME ["/data"]

ENTRYPOINT ["/app/nutella-tracker"]
