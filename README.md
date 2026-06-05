# 🍫 nutella-tracker

My very own Nutella usage tracker & estimator – eine kleine Go-Webanwendung, die
den Verbrauch eines Nutella-Glases trackt, eine Burnrate berechnet und schätzt,
ob das Glas bis zum Zieldatum reicht.

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

| Variable    | Default              | Beschreibung                  |
| ----------- | -------------------- | ----------------------------- |
| `PORT`      | `8080`               | HTTP-Port                     |
| `DATA_FILE` | `data/nutella.json`  | Pfad zur JSON-Datendatei      |

## Tests

```sh
go test ./...
```

## Docker

Image bauen und starten:

```sh
docker compose up --build
```

Die Daten liegen im benannten Volume `nutella-data` (gemountet unter `/data`)
und überleben einen Container-Neustart.

Alternativ das per GitHub Actions veröffentlichte Image verwenden – dazu in der
[docker-compose.yml](docker-compose.yml) `build: .` durch die `image:`-Zeile
ersetzen.

## CI/CD

[GitHub Actions](.github/workflows/ci.yml) baut und testet den Go-Code und
veröffentlicht bei Pushes auf `main`/Tags ein Multi-Stage-Docker-Image nach
GHCR (`ghcr.io/<owner>/nutella-tracker`).

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
