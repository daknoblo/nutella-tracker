"use strict";

// Nutella Tracker – clientseitige Logik: lädt Daten von der API, rendert
// Kennzahlen, Diagramme (Chart.js) und verarbeitet die Formulare.

const api = {
  async listJars() {
    return fetchJSON("/api/jars");
  },
  async createJar(body) {
    return fetchJSON("/api/jars", { method: "POST", body });
  },
  async updateJar(id, body) {
    return fetchJSON(`/api/jars/${encodeURIComponent(id)}`, { method: "PUT", body });
  },
  async deleteJar(id) {
    return fetchJSON(`/api/jars/${encodeURIComponent(id)}`, { method: "DELETE" });
  },
  async activateJar(id) {
    return fetchJSON(`/api/jars/${encodeURIComponent(id)}/activate`, { method: "POST" });
  },
  async addMeasurement(id, body) {
    return fetchJSON(`/api/jars/${encodeURIComponent(id)}/measurements`, { method: "POST", body });
  },
  async deleteMeasurement(id, index) {
    return fetchJSON(`/api/jars/${encodeURIComponent(id)}/measurements/${index}`, { method: "DELETE" });
  },
};

// fetchJSON kapselt fetch inkl. JSON-Serialisierung und Fehlerbehandlung.
async function fetchJSON(url, opts = {}) {
  const init = { method: opts.method || "GET", headers: {} };
  if (opts.body !== undefined) {
    init.headers["Content-Type"] = "application/json";
    init.body = JSON.stringify(opts.body);
  }
  const res = await fetch(url, init);
  if (res.status === 204) return null;
  const data = await res.json().catch(() => null);
  if (!res.ok) {
    throw new Error((data && data.error) || `Fehler ${res.status}`);
  }
  return data;
}

// --- Zustand ---
let jars = [];          // Liste aller jarView-Objekte
let selectedId = null;  // aktuell angezeigtes Glas
let editing = false;    // Einstellungen im Bearbeiten-Modus?
let contentChart = null;
let consumptionChart = null;

// --- DOM-Referenzen ---
const el = (id) => document.getElementById(id);

// --- Formatierung ---
const fmtG = (v) => `${(v ?? 0).toLocaleString("de-DE", { maximumFractionDigits: 1 })} g`;
const fmtDate = (s) => (s ? new Date(s + "T00:00:00").toLocaleDateString("de-DE") : "–");
const todayISO = () => new Date().toISOString().slice(0, 10);

function showToast(msg, isError = false) {
  const t = el("toast");
  t.textContent = msg;
  t.classList.toggle("error", isError);
  t.classList.remove("hidden");
  setTimeout(() => t.classList.add("hidden"), 3000);
}

// --- Laden & Rendern ---
async function reload(preferId) {
  try {
    jars = await api.listJars();
  } catch (e) {
    showToast(e.message, true);
    return;
  }

  const hasJars = jars.length > 0;
  el("emptyHint").classList.toggle("hidden", hasJars);

  // Auswahl bestimmen: bevorzugte ID, bisheriges, sonst aktives, sonst erstes.
  if (preferId && jars.some((j) => j.jar.id === preferId)) {
    selectedId = preferId;
  } else if (!jars.some((j) => j.jar.id === selectedId)) {
    const active = jars.find((j) => j.active);
    selectedId = active ? active.jar.id : hasJars ? jars[0].jar.id : null;
  }

  renderJarSelect();
  renderSelected();
}

function renderJarSelect() {
  const sel = el("jarSelect");
  sel.innerHTML = "";
  for (const v of jars) {
    const opt = document.createElement("option");
    opt.value = v.jar.id;
    const name = v.jar.name || v.jar.id;
    opt.textContent = v.active ? `${name} (aktiv)` : name;
    if (v.jar.id === selectedId) opt.selected = true;
    sel.appendChild(opt);
  }
}

function currentView() {
  return jars.find((j) => j.jar.id === selectedId) || null;
}

function renderSelected() {
  const view = currentView();
  const show = !!view && !editing;

  el("statsSection").classList.toggle("hidden", !show);
  el("chartsSection").classList.toggle("hidden", !show);
  el("measureSection").classList.toggle("hidden", !show);
  // Einstellungen sind sichtbar, wenn ein Glas gewählt ODER neu angelegt wird.
  el("settingsSection").classList.toggle("hidden", !view && !editing);

  if (!view) {
    if (editing) renderSettingsForm(null);
    return;
  }

  renderStats(view);
  renderCharts(view);
  renderMeasurements(view);
  renderSettingsForm(view);
}

function renderStats(view) {
  const s = view.stats;
  el("jarName").textContent = view.jar.name ? `– ${view.jar.name}` : "";

  const items = [
    { label: "Aktueller Inhalt", value: fmtG(s.currentNet) },
    { label: "Anfangsinhalt", value: fmtG(s.initialNet) },
    { label: "Gesamt verbraucht", value: fmtG(s.totalConsumed) },
    { label: "Seit letzter Messung", value: fmtG(s.consumedSinceLast) },
    { label: "Burnrate / Tag", value: fmtG(s.burnRatePerDay) },
    { label: "Burnrate / Esstag (Sa/So)", value: fmtG(s.burnRatePerEatingDay) },
    { label: "Soll-Verbrauch / Tag", value: fmtG(s.plannedDailyRate) },
    { label: "Ø Verbrauch / Messung", value: fmtG(s.avgConsumed) },
    { label: "Max / Min Verbrauch", value: `${fmtG(s.maxConsumed)} / ${fmtG(s.minConsumed)}` },
    { label: "Geschätzt leer am", value: fmtDate(s.estimatedEmptyDate) },
  ];

  el("statsGrid").innerHTML = items
    .map((i) => `<div class="stat"><div class="value">${i.value}</div><div class="label">${i.label}</div></div>`)
    .join("");

  renderTargetBanner(view);
}

function renderTargetBanner(view) {
  const s = view.stats;
  const banner = el("targetBanner");
  const target = fmtDate(view.jar.targetDate);
  let cls = "unknown";
  let text = "Noch zu wenige Messungen für eine Schätzung.";

  const diff = Math.abs(s.targetDiffDays);
  if (s.targetStatus === "ja") {
    cls = "ok";
    text = `✓ Das Glas reicht bis zum Zieldatum (${target}) – ca. ${diff} Tage Puffer.`;
  } else if (s.targetStatus === "knapp") {
    cls = "warn";
    text = `≈ Das Glas reicht knapp bis zum Zieldatum (${target}).`;
  } else if (s.targetStatus === "nein") {
    cls = "bad";
    text = `✗ Das Glas reicht voraussichtlich NICHT bis zum Zieldatum (${target}) – ca. ${diff} Tage zu früh leer.`;
  }
  banner.className = `banner ${cls}`;
  banner.textContent = text;
}

function renderMeasurements(view) {
  const tbody = el("measureTable").querySelector("tbody");
  tbody.innerHTML = "";
  const jar = view.jar;
  const measurements = (jar.measurements || [])
    .map((m, i) => ({ ...m, _index: i }))
    .sort((a, b) => a.date.localeCompare(b.date));

  let prevNet = jar.grossFullWeight - jar.tareWeight;
  for (const m of measurements) {
    const net = m.grossWeight - jar.tareWeight;
    const consumed = prevNet - net;
    prevNet = net;

    const tr = document.createElement("tr");
    tr.innerHTML =
      `<td>${fmtDate(m.date)}</td>` +
      `<td>${fmtG(m.grossWeight)}</td>` +
      `<td>${fmtG(net)}</td>` +
      `<td>${fmtG(consumed)}</td>` +
      `<td><button data-index="${m._index}" class="danger">✕</button></td>`;
    tr.querySelector("button").addEventListener("click", () => onDeleteMeasurement(m._index));
    tbody.appendChild(tr);
  }
  if (measurements.length === 0) {
    tbody.innerHTML = `<tr><td colspan="5" class="muted">Noch keine Messungen.</td></tr>`;
  }
}

// --- Diagramme ---
function renderCharts(view) {
  const jar = view.jar;
  const s = view.stats;

  // Ist-Kurve: Start (voll) + alle Messungen.
  const points = [{ x: jar.startDate, y: s.initialNet }];
  for (const c of s.consumption) {
    points.push({ x: c.date, y: c.net });
  }

  // Soll-Linie: linear von Start (initialNet) bis Zieldatum (0).
  const sollLine = [
    { x: jar.startDate, y: s.initialNet },
    { x: jar.targetDate, y: 0 },
  ];

  // Prognose-Linie: von letzter Messung bis geschätztem Leerdatum (0).
  let prognose = [];
  if (s.estimatedEmptyDate && points.length > 1) {
    const last = points[points.length - 1];
    prognose = [
      { x: last.x, y: last.y },
      { x: s.estimatedEmptyDate, y: 0 },
    ];
  }

  contentChart = upsertChart(contentChart, "contentChart", {
    type: "line",
    data: {
      datasets: [
        {
          label: "Ist-Restinhalt",
          data: points,
          borderColor: "#b9722e",
          backgroundColor: "rgba(185,114,46,0.15)",
          tension: 0.2,
          fill: true,
        },
        {
          label: "Soll (linear)",
          data: sollLine,
          borderColor: "#2e7d32",
          borderDash: [6, 4],
          pointRadius: 0,
          fill: false,
        },
        {
          label: "Prognose",
          data: prognose,
          borderColor: "#c62828",
          borderDash: [2, 3],
          pointRadius: 0,
          fill: false,
        },
      ],
    },
    options: {
      responsive: true,
      scales: {
        x: { type: "time", time: { unit: "day" }, adapters: {} },
        y: { beginAtZero: true, title: { display: true, text: "Gramm" } },
      },
      plugins: { legend: { position: "bottom" } },
    },
  });

  // Verbrauch pro Messung (Balken).
  const labels = s.consumption.map((c) => fmtDate(c.date));
  const consumed = s.consumption.map((c) => round1(c.consumed));

  consumptionChart = upsertChart(consumptionChart, "consumptionChart", {
    type: "bar",
    data: {
      labels,
      datasets: [
        {
          label: "Verbrauch (g)",
          data: consumed,
          backgroundColor: "#6d4326",
        },
      ],
    },
    options: {
      responsive: true,
      scales: { y: { beginAtZero: true, title: { display: true, text: "Gramm" } } },
      plugins: { legend: { display: false } },
    },
  });
}

// upsertChart zerstört ggf. ein bestehendes Chart und erstellt es neu.
// (Zeitachse nutzt kategoriale Labels als Fallback, falls kein Time-Adapter.)
function upsertChart(existing, canvasId, config) {
  if (existing) existing.destroy();
  // Ohne Date-Adapter kann Chart.js keine "time"-Achse rendern -> auf
  // kategoriale x-Achse mit formatierten Datums-Labels zurückfallen.
  if (config.options?.scales?.x?.type === "time") {
    for (const ds of config.data.datasets) {
      ds.data = ds.data.map((p) => ({ x: fmtDate(p.x), y: p.y }));
    }
    config.options.scales.x = { title: { display: true, text: "Datum" } };
  }
  return new Chart(el(canvasId).getContext("2d"), config);
}

const round1 = (v) => Math.round((v ?? 0) * 10) / 10;

// --- Einstellungen-Formular ---
function renderSettingsForm(view) {
  const title = el("settingsSection").querySelector("h2");
  const deleteBtn = el("deleteJarBtn");
  const cancelBtn = el("jarFormCancel");

  if (view) {
    const j = view.jar;
    el("jarId").value = j.id;
    el("jarFormName").value = j.name || "";
    el("jarFormGross").value = j.grossFullWeight;
    el("jarFormTare").value = j.tareWeight;
    el("jarFormStart").value = j.startDate;
    el("jarFormTarget").value = j.targetDate;
    title.textContent = "Einstellungen";
    deleteBtn.classList.remove("hidden");
    cancelBtn.classList.add("hidden");
  } else {
    // Neues Glas: sinnvolle Defaults.
    el("jarId").value = "";
    el("jarFormName").value = "";
    el("jarFormGross").value = 1200;
    el("jarFormTare").value = 200;
    el("jarFormStart").value = todayISO();
    el("jarFormTarget").value = addMonthsISO(todayISO(), 2);
    title.textContent = "Neues Glas anlegen";
    deleteBtn.classList.add("hidden");
    cancelBtn.classList.remove("hidden");
  }
}

function addMonthsISO(iso, months) {
  const d = new Date(iso + "T00:00:00");
  d.setMonth(d.getMonth() + months);
  return d.toISOString().slice(0, 10);
}

// --- Event-Handler ---
async function onSaveJar(e) {
  e.preventDefault();
  const id = el("jarId").value;
  const body = {
    name: el("jarFormName").value.trim(),
    grossFullWeight: parseFloat(el("jarFormGross").value),
    tareWeight: parseFloat(el("jarFormTare").value),
    startDate: el("jarFormStart").value,
    targetDate: el("jarFormTarget").value,
  };
  try {
    let result;
    if (id) {
      result = await api.updateJar(id, body);
    } else {
      result = await api.createJar(body);
    }
    editing = false;
    showToast("Glas gespeichert.");
    await reload(result.jar.id);
  } catch (err) {
    showToast(err.message, true);
  }
}

async function onAddMeasurement(e) {
  e.preventDefault();
  const view = currentView();
  if (!view) return;
  const body = {
    date: el("measureDate").value || todayISO(),
    grossWeight: parseFloat(el("measureWeight").value),
  };
  try {
    await api.addMeasurement(view.jar.id, body);
    el("measureWeight").value = "";
    showToast("Messung gespeichert.");
    await reload(view.jar.id);
  } catch (err) {
    showToast(err.message, true);
  }
}

async function onDeleteMeasurement(index) {
  const view = currentView();
  if (!view) return;
  if (!confirm("Diese Messung wirklich löschen?")) return;
  try {
    await api.deleteMeasurement(view.jar.id, index);
    showToast("Messung gelöscht.");
    await reload(view.jar.id);
  } catch (err) {
    showToast(err.message, true);
  }
}

async function onDeleteJar() {
  const view = currentView();
  if (!view) return;
  if (!confirm(`Glas "${view.jar.name || view.jar.id}" inkl. Historie löschen?`)) return;
  try {
    await api.deleteJar(view.jar.id);
    selectedId = null;
    showToast("Glas gelöscht.");
    await reload();
  } catch (err) {
    showToast(err.message, true);
  }
}

async function onActivate() {
  const view = currentView();
  if (!view) return;
  try {
    await api.activateJar(view.jar.id);
    showToast("Aktives Glas gesetzt.");
    await reload(view.jar.id);
  } catch (err) {
    showToast(err.message, true);
  }
}

function onNewJar() {
  editing = true;
  selectedId = null;
  renderJarSelect();
  renderSelected();
}

function onCancelEdit() {
  editing = false;
  reload();
}

// --- Initialisierung ---
function init() {
  el("measureDate").value = todayISO();

  el("jarSelect").addEventListener("change", (e) => {
    editing = false;
    selectedId = e.target.value;
    renderSelected();
  });
  el("jarForm").addEventListener("submit", onSaveJar);
  el("measureForm").addEventListener("submit", onAddMeasurement);
  el("deleteJarBtn").addEventListener("click", onDeleteJar);
  el("activateBtn").addEventListener("click", onActivate);
  el("newJarBtn").addEventListener("click", onNewJar);
  el("jarFormCancel").addEventListener("click", onCancelEdit);

  reload();
}

document.addEventListener("DOMContentLoaded", init);
