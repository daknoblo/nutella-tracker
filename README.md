# 🍫 nutella-tracker

Tracks a Nutella jar by weight, calculates the burn rate and estimates whether it lasts until your target date — with optional photo-based scale reading.

## Features

- Mehrere Gläser nacheinander mit eigener Mess-Historie (ein aktives Glas)
- Wiege-Messungen (Bruttogewicht in g) → Nutzinhalt = Brutto − Tara
- Kennzahlen: aktueller Inhalt, gesamt verbraucht, Verbrauch seit letzter Messung
- Burnrate pro Tag **und** pro Esstag (Sa/So)
- Reichweiten-Schätzung (voraussichtliches Leerdatum) und Soll/Ist-Abgleich
  zum Zieldatum (ja / knapp / nein)
- Diagramme (Chart.js): Restinhalt-Verlauf mit Soll-Linie + Prognose,
  Verbrauch pro Messung
- Konfigurierbar: Glasgröße (Brutto/Tara), Start- und Zieldatum
- Persistenz in einer einzelnen JSON-Datei (atomares Schreiben), keine Datenbank
- Keine Authentifizierung – rein für den privaten Einsatz

## Technik

- **Go** (Standardbibliothek, HTTP-Server mit `net/http`)
- **Web-UI** als eingebettete statische Dateien (`embed`)
- **Diagramme** clientseitig per Chart.js (CDN)

## Lokal starten

```sh
go run ./cmd/server
# danach: http://localhost:8080
```

Konfiguration über Umgebungsvariablen:

| Variable                   | Default                | Beschreibung                                        |
| -------------------------- | ---------------------- | --------------------------------------------------- |
| `PORT`                     | `8080`                 | HTTP-Port                                           |
| `DATA_FILE`                | `data/nutella.json`    | Pfad zur JSON-Datendatei                            |
| `AZURE_VISION_ENDPOINT`    | –                      | Foundry-/Azure-OpenAI-Endpoint (Foto-Erkennung)     |
| `AZURE_VISION_KEY`         | –                      | Secret/API-Key (nur via Env, nie in der JSON)       |
| `AZURE_VISION_MODEL`       | –                      | Deployment-/Modellname (z. B. `gpt-4o`)             |
| `AZURE_VISION_API_VERSION` | `2024-08-01-preview`   | API-Version des Vision-Endpoints                    |

## Foto-Erkennung (optional)

Statt das Gewicht manuell einzutippen, kann ein Foto des Waagen-Displays
aufgenommen werden – ein Vision-Modell in **Microsoft Foundry** liest den Wert
aus. Die Funktion erscheint nur, wenn die `AZURE_VISION_*`-Variablen gesetzt
sind. **Die Bilder werden nicht gespeichert**; das Foto wird ausschließlich für
die einzelne Erkennungs-Anfrage an das Modell übermittelt.

Ablauf: Auf dem Handy „📷 Foto aufnehmen & auslesen" tippen → das Foto geht an
das Backend → das Backend ruft das Vision-Modell auf → der erkannte Wert wird
ins Mess-Formular eingetragen und kann nach Prüfung gespeichert werden.

### Einrichtung in Microsoft Foundry

1. In Microsoft Foundry / Azure OpenAI ein **vision-fähiges Modell** (z. B.
   `gpt-4o` oder `gpt-4o-mini`) als **Deployment** anlegen.
2. Folgende drei Werte aus dem Portal übernehmen:
   - **Endpoint-URL** → `AZURE_VISION_ENDPOINT`
     (Form: `https://<ressource>.openai.azure.com`)
   - **Secret/API-Key** → `AZURE_VISION_KEY`
   - **Deployment-/Modellname** → `AZURE_VISION_MODEL`
3. Optional die **API-Version** anpassen (`AZURE_VISION_API_VERSION`).

Lokal lassen sich die Werte über eine `.env`-Datei bereitstellen
(siehe [.env.example](.env.example)); diese ist per `.gitignore` ausgeschlossen.

## Tests

```sh
go test ./...
```

## Docker

Image bauen und starten:

```sh
docker compose up --build
```

Die Daten liegen im benannten Volume `nutella-data` (gemountet unter `/appdata`)
und überleben einen Container-Neustart.

Alternativ das per GitHub Actions veröffentlichte Image verwenden – dazu in der
[docker-compose.yml](docker-compose.yml) `build: .` durch die `image:`-Zeile
ersetzen oder [docker-compose.example.yml](docker-compose.example.yml) als
gehärtete Vorlage nutzen.

## CI/CD

[CI](.github/workflows/ci.yml) prüft Formatierung, Vet, Lint, Vulnerabilities,
Tests und Build. [Release](.github/workflows/release.yml) veröffentlicht bei
Pushes auf `main`/`develop` und Tags ein signiertes Multi-Arch-Docker-Image nach
GHCR (`ghcr.io/<owner>/nutella-tracker`) und lädt Scan-Ergebnisse hoch.

## API (Auszug)

| Methode & Pfad                              | Zweck                          |
| ------------------------------------------- | ------------------------------ |
| `GET /api/jars`                             | Alle Gläser inkl. Statistik    |
| `POST /api/jars`                            | Neues Glas anlegen             |
| `PUT /api/jars/{id}`                        | Glas-Stammdaten ändern         |
| `DELETE /api/jars/{id}`                     | Glas löschen                   |
| `POST /api/jars/{id}/activate`              | Glas aktiv setzen              |
| `GET /api/jars/{id}/stats`                  | Statistik eines Glases         |
| `POST /api/jars/{id}/measurements`          | Messung hinzufügen             |
| `DELETE /api/jars/{id}/measurements/{idx}`  | Messung löschen                |
| `GET /api/active`                           | Aktives Glas inkl. Statistik   |
| `GET /api/config`                           | Verfügbare Features (z. B. Vision) |
| `POST /api/vision/recognize`                | Foto (multipart `photo`) → erkanntes Gewicht |


## Lizenz

Veröffentlicht unter der [MIT-Lizenz](LICENSE).
