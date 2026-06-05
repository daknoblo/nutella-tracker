# Nutella Tracker – Projektanforderungen & Instruktionen

Dieses Dokument hält alle Anforderungen an den **Nutella Tracker** fest. Es dient als
verbindliche Referenz für die Implementierung. Antworten/Code bitte auf Deutsch
kommentieren, Code-Bezeichner auf Englisch.

---

## 1. Ziel der Anwendung

Eine kleine Web-Anwendung, mit der der Verbrauch bzw. der Inhalt eines Nutella-Glases
getrackt und geschätzt wird.

Beispiel-Szenario:
- Start mit einem 1-kg-Glas Nutella, das **2 Monate** reichen soll.
- Nutella wird nur am Wochenende gegessen (Samstag/Sonntag).
- Am Wochenende wird das Glas gewogen und das **aktuelle Gewicht** eingetragen und gespeichert.
- Daraus wird der **Nutzinhalt** berechnet (Füllgewicht abzüglich Leergewicht des Glases).
- Daraus wird berechnet, **wie viel pro Mess-/Esstag** verbraucht wurde
  (z. B. 15 g am Samstag, 15 g am Sonntag).
- Es wird eine **Burnrate** (Verbrauch pro Zeit) berechnet und geschätzt, **ob das Glas
  die geplanten 2 Monate reicht**.

---

## 2. Technologie-Stack

- **Sprache:** Go (Standardbibliothek bevorzugt; minimale, gut gepflegte Dependencies erlaubt).
- **Persistenz:** Eine einfache **JSON-Datei** als Datenspeicher (keine externe Datenbank).
  - Datei-Pfad konfigurierbar (z. B. über Env-Variable), Default in einem `data/`-Verzeichnis.
  - Schreiben atomar (temporäre Datei + Rename), um Korruption zu vermeiden.
- **UI:** **Web-UI im Browser** mit Diagrammen.
- **Diagramme:** **clientseitig per JavaScript-Charts** (z. B. Chart.js), Daten kommen vom Go-Backend.
- **Auth:** **Keine Authentifizierung** – die App läuft rein privat.

---

## 3. Fachliche Anforderungen

### 3.1 Datenmodell (mehrere Gläser mit Historie)

- Es können **mehrere Gläser nacheinander** getrackt werden, jeweils mit eigener Historie.
- Es gibt jeweils ein **aktives Glas**; ältere Gläser bleiben als Historie erhalten.
- Pro Glas:
  - eindeutige ID
  - Name/Bezeichnung (optional)
  - **Bruttofüllgewicht** (Glas voll) bzw. Nennfüllmenge des Inhalts
  - **Leergewicht des Glases** (Tara), um den Nutzinhalt zu berechnen
  - **Startdatum**
  - **Zieldatum** / geplante Haltbarkeitsdauer (z. B. 2 Monate)
  - Liste von **Messungen**

### 3.2 Messung (Weigh-In)

- Felder pro Messung:
  - Datum (Default: heute)
  - **aktuelles Bruttogewicht** (Glas + Restinhalt) in **Gramm (g)**
- **Einheit:** Gewichte werden in **Gramm (g)** eingetragen.
- Abgeleitet pro Messung:
  - aktueller Nutzinhalt = aktuelles Bruttogewicht − Leergewicht
  - verbrauchte Menge seit letzter Messung
  - Verbrauch pro Tag/pro Esstag

### 3.3 Berechnungen & Statistiken

- **Aktueller Nutzinhalt** (verbleibendes Nutella in g).
- **Bereits verbraucht** (gesamt und seit letzter Messung).
- **Burnrate**: durchschnittlicher Verbrauch (z. B. pro Tag und/oder pro Wochenende/Esstag).
- **Reichweiten-Schätzung**: voraussichtliches Leerdatum bei aktueller Burnrate.
- **Soll/Ist-Abgleich**: Reicht das Glas bis zum Zieldatum? (ja/knapp/nein + Differenz).
- Weitere sinnvolle Statistiken über den Verbrauch (z. B. Verbrauch pro Woche,
  Durchschnitt, größter/kleinster Verbrauch).

### 3.4 Diagramme

- **Verlauf des Restinhalts** über die Zeit (Ist-Kurve).
- **Soll-Linie** (idealer linearer Verbrauch vom Start bis zum Zieldatum) zum Vergleich.
- **Verbrauch pro Messung/Woche** (z. B. Balkendiagramm).
- Optional: Prognose-/Trendlinie bis zum geschätzten Leerdatum.

### 3.5 Einstellungen / Konfiguration

In der App konfigurierbar:
- **Glasgröße** (Bruttofüllgewicht und/oder Tara/Leergewicht).
- **Startdatum**.
- **Zieldatum** bzw. geplante Haltbarkeitsdauer (z. B. 2 Monate).

---

## 4. Architektur & Struktur (Richtwerte)

- Go-Webserver, der sowohl die statische Web-UI als auch eine kleine JSON-API ausliefert.
- Klare Trennung:
  - Domain/Logik (Berechnungen, Burnrate, Schätzungen)
  - Persistenz (Laden/Speichern der JSON-Datei)
  - HTTP-Handler / API
  - Web-UI (HTML/JS/CSS, Charts)
- Berechnungslogik soll **unit-getestet** sein (Go-Tests).

---

## 5. Deployment

- Die Anwendung wird in einen **Docker-Container** gebaut.
- Build erfolgt **per Pipeline mit GitHub Actions** (Image bauen und veröffentlichen).
- Ziel: Das Image kann später auf dem eigenen **Docker-Host via Docker Compose** geladen
  und betrieben werden.
- Anforderungen an das Image:
  - kleines, mehrstufiges Build-Image (Multi-Stage: Build in Go-Image, Runtime in schlankem Base-Image)
  - JSON-Datenverzeichnis als **Volume** persistierbar (Daten überleben Container-Neustart)
  - Port konfigurierbar (Env-Variable), sinnvoller Default
- Es soll eine **`docker-compose.yml`** bereitstehen, die Image + Volume + Port mappt.

---

## 6. Nicht-Ziele / bewusste Vereinfachungen

- Keine externe Datenbank.
- Keine Benutzerverwaltung / kein Login.
- Kein Multi-User-Betrieb (rein privat).

---

## 7. Offene Punkte / Annahmen

- Falls Bruttofüllgewicht und Tara unbekannt sind, werden sinnvolle Defaults vorgeschlagen
  (z. B. 1000 g Inhalt, typisches Glas-Leergewicht) und sind in den Einstellungen änderbar.
