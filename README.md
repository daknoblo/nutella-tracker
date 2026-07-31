# 🍫 nutella-tracker

[![CI](https://github.com/daknoblo/nutella-tracker/actions/workflows/ci.yml/badge.svg)](https://github.com/daknoblo/nutella-tracker/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/daknoblo/nutella-tracker)](https://github.com/daknoblo/nutella-tracker/releases/latest)
[![Go](https://img.shields.io/github/go-mod/go-version/daknoblo/nutella-tracker)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![GHCR](https://img.shields.io/badge/ghcr.io-nutella--tracker-blue?logo=docker)](https://github.com/daknoblo/nutella-tracker/pkgs/container/nutella-tracker)

Tracks a Nutella jar by weight, calculates the burn rate and estimates whether it lasts until your target date — with optional photo-based scale reading.

> **Note on language:** the user interface is German. Code, comments and this
> documentation are English.

## Features

- Several jars one after another, each with its own measurement history
  (exactly one active jar)
- Weigh-ins (gross weight in g) → net contents = gross − tare
- Key figures: current contents, total consumed, consumption since the last
  weigh-in
- Burn rate per day **and** per eating day (Sat/Sun)
- Range estimate (projected empty date) and a target comparison against the
  target date (yes / close / no)
- Charts (Chart.js): remaining contents over time with an ideal line and a
  forecast, plus consumption per weigh-in
- Configurable: jar size (gross/tare), start date and target date
- Persisted in a single JSON file (written atomically), no database
- No authentication — strictly for private use

## Technology

- **Go** (standard library, HTTP server via `net/http`)
- **Web UI** as embedded static files (`embed`)
- **Charts** rendered client-side with Chart.js (CDN)

## Running locally

```sh
go run ./cmd/server
# then open http://localhost:8080
```

Configuration via environment variables:

| Variable                   | Default                | Description                                          |
| -------------------------- | ---------------------- | ---------------------------------------------------- |
| `PORT`                     | `8080`                 | HTTP port                                            |
| `DATA_FILE`                | `data/nutella.json`    | Path to the JSON data file                           |
| `AZURE_VISION_ENDPOINT`    | –                      | Foundry / Azure OpenAI endpoint (photo recognition)  |
| `AZURE_VISION_KEY`         | –                      | Secret / API key (via env only, never in the JSON)   |
| `AZURE_VISION_MODEL`       | –                      | Deployment / model name (e.g. `gpt-4o`)              |
| `AZURE_VISION_API_VERSION` | `2024-08-01-preview`   | API version of the vision endpoint                   |

## Photo recognition (optional)

Instead of typing the weight by hand, you can take a photo of the scale's
display — a vision model in **Microsoft Foundry** reads the value from it. The
feature only appears when the `AZURE_VISION_*` variables are set. **Images are
never stored**; the photo is sent to the model for that single recognition
request only.

Flow: tap "📷 Take photo & read" on your phone → the photo goes to the backend →
the backend calls the vision model → the recognised value is filled into the
weigh-in form, where you can check it before saving.

### Setting it up in Microsoft Foundry

1. In Microsoft Foundry / Azure OpenAI, create a **vision capable model**
   (e.g. `gpt-4o` or `gpt-4o-mini`) as a **deployment**.
2. Take these three values from the portal:
   - **Endpoint URL** → `AZURE_VISION_ENDPOINT`
     (form: `https://<resource>.openai.azure.com`)
   - **Secret / API key** → `AZURE_VISION_KEY`
   - **Deployment / model name** → `AZURE_VISION_MODEL`
3. Optionally adjust the **API version** (`AZURE_VISION_API_VERSION`).

Locally these values can be provided through a `.env` file
(see [.env.example](.env.example)); it is excluded via `.gitignore`.

## Tests

```sh
go test ./...
```

## Docker

Build and start the image:

```sh
docker compose up --build
```

The data lives in the named volume `nutella-data` (mounted at `/appdata`) and
survives a container restart.

Alternatively use the image published by GitHub Actions — either replace
`build: .` in [docker-compose.yml](docker-compose.yml) with the `image:` line,
or use [docker-compose.example.yml](docker-compose.example.yml) as a hardened
template.

## CI/CD

[CI](.github/workflows/ci.yml) checks formatting, vet, lint, vulnerabilities,
tests and the build. [Release](.github/workflows/release.yml) publishes a signed
multi-arch Docker image to GHCR (`ghcr.io/<owner>/nutella-tracker`) on pushes to
`main`/`develop` and on tags, and uploads the scan results.

## API (excerpt)

| Method & path                               | Purpose                                       |
| ------------------------------------------- | --------------------------------------------- |
| `GET /api/jars`                             | All jars including statistics                 |
| `POST /api/jars`                            | Create a new jar                              |
| `PUT /api/jars/{id}`                        | Change a jar's master data                    |
| `DELETE /api/jars/{id}`                     | Delete a jar                                  |
| `POST /api/jars/{id}/activate`              | Mark a jar as active                          |
| `GET /api/jars/{id}/stats`                  | Statistics for a single jar                   |
| `POST /api/jars/{id}/measurements`          | Add a weigh-in                                |
| `DELETE /api/jars/{id}/measurements/{idx}`  | Delete a weigh-in                             |
| `GET /api/active`                           | Active jar including statistics               |
| `GET /api/config`                           | Available features (e.g. vision)              |
| `POST /api/vision/recognize`                | Photo (multipart `photo`) → recognised weight |


## License

Released under the [MIT License](LICENSE).
