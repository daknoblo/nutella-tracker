# syntax=docker/dockerfile:1

# --- Build-Stage: Go-Binary statisch kompilieren ---
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /src

# Abhängigkeiten zuerst (besseres Layer-Caching).
COPY go.mod ./
# Falls später go.sum hinzukommt, wird es mitkopiert.
COPY go.su[m] ./
RUN go mod download

# Restlichen Quellcode kopieren und bauen.
COPY . .
# CGO aus -> statisches Binary, klein und ohne libc-Abhängigkeit.
RUN mkdir -p /out /appdata && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" \
    -o /out/nutella-tracker ./cmd/server

# --- Runtime-Stage: schlankes, gehärtetes Image ---
FROM gcr.io/distroless/static:nonroot AS runtime

WORKDIR /app
COPY --from=build /out/nutella-tracker /app/nutella-tracker
COPY --from=build --chown=65532:65532 /appdata /appdata

# Daten landen im Volume /appdata.
ENV PORT=8080 \
    DATA_FILE=/appdata/nutella.json

EXPOSE 8080
VOLUME ["/appdata"]

USER nonroot:nonroot

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/app/nutella-tracker", "-healthcheck"]

ENTRYPOINT ["/app/nutella-tracker"]
