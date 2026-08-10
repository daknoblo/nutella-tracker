"use strict";

// Demo-Modus: ersetzt die Server-API durch einen statischen Snapshot, damit die
// komplette Web-UI ohne Backend läuft (z. B. auf GitHub Pages). Schreibende
// Aufrufe sind bewusst deaktiviert – die Demo ist für alle Besucher gleich.
(function () {
  const SNAPSHOT_URL = "api/jars.json";
  const READ_ONLY_MESSAGE =
    "Demo-Modus: Diese Seite ist schreibgeschützt. Lade dir den Tracker herunter, um eigene Gläser zu tracken.";

  const realFetch = window.fetch.bind(window);
  let snapshotPromise = null;

  // Diagramm-Animationen abschalten: die Demo wirkt dadurch schneller und die
  // automatisierten Screenshots werden reproduzierbar.
  if (window.Chart) {
    window.Chart.defaults.animation = false;
  }

  function jsonResponse(body, status = 200) {
    return new Response(JSON.stringify(body), {
      status,
      headers: { "Content-Type": "application/json; charset=utf-8" },
    });
  }

  function loadSnapshot() {
    if (!snapshotPromise) {
      snapshotPromise = realFetch(SNAPSHOT_URL).then((res) => {
        if (!res.ok) throw new Error(`Demo-Daten konnten nicht geladen werden (${res.status})`);
        return res.json();
      });
    }
    return snapshotPromise;
  }

  // simulateRecognition liefert einen plausiblen Gewichtswert, damit die
  // optionale Foto-Erkennung auch ohne Vision-Backend vorführbar ist.
  function simulateRecognition(views) {
    const view = views.find((v) => v.active) || views[0];
    const measurements = (view && view.jar.measurements) || [];
    const last = measurements[measurements.length - 1];
    const base = last ? last.grossWeight : 1200;
    return Math.round((base - 55) * 10) / 10;
  }

  async function route(path, method) {
    const views = await loadSnapshot();

    if (path === "/api/vision/recognize") {
      return jsonResponse({ grossWeight: simulateRecognition(views) });
    }
    if (method !== "GET") {
      return jsonResponse({ error: READ_ONLY_MESSAGE }, 403);
    }
    if (path === "/api/config") {
      return jsonResponse({ visionEnabled: true });
    }
    if (path === "/api/jars") {
      return jsonResponse(views);
    }
    if (path === "/api/active") {
      const active = views.find((v) => v.active);
      return active
        ? jsonResponse(active)
        : jsonResponse({ error: "Glas nicht gefunden" }, 404);
    }

    const match = path.match(/^\/api\/jars\/([^/]+)(\/stats)?$/);
    if (match) {
      const view = views.find((v) => v.jar.id === decodeURIComponent(match[1]));
      if (!view) return jsonResponse({ error: "Glas nicht gefunden" }, 404);
      return jsonResponse(match[2] ? view.stats : view);
    }

    return jsonResponse({ error: `Unbekannter Endpunkt ${path}` }, 404);
  }

  window.fetch = function (input, init = {}) {
    const url = typeof input === "string" ? input : input.url;
    const path = new URL(url, window.location.href).pathname;
    if (!path.startsWith("/api/")) {
      return realFetch(input, init);
    }
    const method = (init.method || (typeof input === "object" && input.method) || "GET").toUpperCase();
    return route(path, method).catch((err) => jsonResponse({ error: err.message }, 500));
  };

  // Rückfragen (z. B. "Messung löschen?") würden die Demo nur blockieren.
  window.confirm = () => true;
})();
