// Nimmt automatisiert Screenshots der statischen Demo-Seite auf.
//
// Die Aufnahmeliste stammt aus <dist>/screenshots.json, die von `go run
// ./cmd/gensite` erzeugt wird – damit sind Doku, Galerie und Screenshots immer
// synchron. Aufruf: node capture.mjs [--dist ../../dist]

import { chromium } from "playwright";
import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { mkdir } from "node:fs/promises";
import { extname, join, resolve, sep } from "node:path";

const MIME = {
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".json": "application/json; charset=utf-8",
  ".png": "image/png",
  ".svg": "image/svg+xml",
};

// Die Demo lädt Chart.js vom CDN. Für die Aufnahme wird es aus node_modules
// bedient: so sind die Screenshots unabhängig von der CDN-Verfügbarkeit.
const CDN_MIRROR = [
  [/chart\.js@.*chart\.umd/, "chart.js/dist/chart.umd.js"],
  [/chartjs-adapter-date-fns/, "chartjs-adapter-date-fns/dist/chartjs-adapter-date-fns.bundle.js"],
];

const distIndex = process.argv.indexOf("--dist");
const distDir = resolve(distIndex === -1 ? "../../dist" : process.argv[distIndex + 1]);
const shotsDir = join(distDir, "assets", "screenshots");

// Statischer Dateiserver: die Demo lädt ihre Daten per fetch, das funktioniert
// unter file:// nicht.
function startServer(root) {
  const server = createServer(async (req, res) => {
    try {
      const urlPath = decodeURIComponent(new URL(req.url, "http://localhost").pathname);
      let filePath = resolve(join(root, urlPath.endsWith("/") ? urlPath + "index.html" : urlPath));
      if (filePath !== root && !filePath.startsWith(root + sep)) {
        res.writeHead(403).end("forbidden");
        return;
      }
      const body = await readFile(filePath);
      res.writeHead(200, { "Content-Type": MIME[extname(filePath)] ?? "application/octet-stream" });
      res.end(body);
    } catch {
      res.writeHead(404).end("not found");
    }
  });
  return new Promise((ok) => server.listen(0, "127.0.0.1", () => ok(server)));
}

// Wartet, bis die Demo-Daten geladen und alle Diagramme gezeichnet sind.
async function waitForDemo(page) {
  await page.waitForFunction(() => typeof window.Chart === "function", null, { timeout: 30_000 });
  await page.waitForSelector("#statsSection:not(.hidden)");
  await page.waitForFunction(
    () => document.querySelectorAll("#measureTable tbody tr").length > 1
  );
  await page.evaluate(() => document.fonts.ready.then(() => true));
}

// mirrorCDN beantwortet die CDN-Anfragen der Demo aus node_modules.
async function mirrorCDN(page) {
  await page.route("https://cdn.jsdelivr.net/**", async (route) => {
    const url = route.request().url();
    const entry = CDN_MIRROR.find(([pattern]) => pattern.test(url));
    if (!entry) {
      await route.continue();
      return;
    }
    await route.fulfill({
      contentType: MIME[".js"],
      body: await readFile(resolve("node_modules", entry[1])),
    });
  });
}

async function main() {
  const shots = JSON.parse(await readFile(join(distDir, "screenshots.json"), "utf8"));
  await mkdir(shotsDir, { recursive: true });

  const server = await startServer(distDir);
  const baseURL = `http://127.0.0.1:${server.address().port}`;
  // --lang stellt die Browser-Oberfläche (z. B. Datumsfelder) auf Deutsch.
  const browser = await chromium.launch({ args: ["--lang=de-DE"] });

  try {
    for (const shot of shots) {
      const context = await browser.newContext({
        viewport: { width: shot.width, height: shot.height },
        deviceScaleFactor: 2,
        locale: "de-DE",
        timezoneId: "Europe/Berlin",
        reducedMotion: "reduce",
      });
      const page = await context.newPage();
      await mirrorCDN(page);
      await page.goto(`${baseURL}/demo/`, { waitUntil: "networkidle" });
      await waitForDemo(page);

      if (shot.selectJar) {
        await page.selectOption("#jarSelect", shot.selectJar);
      }
      if (shot.click) {
        await page.click(shot.click);
      }

      const path = join(shotsDir, `${shot.name}.png`);
      if (shot.selector) {
        await page.locator(shot.selector).screenshot({ path });
      } else {
        await page.screenshot({ path, fullPage: Boolean(shot.fullPage) });
      }
      console.log(`✓ ${shot.name}.png`);
      await context.close();
    }
  } finally {
    await browser.close();
    server.close();
  }

  console.log(`${shots.length} Screenshots in ${shotsDir}`);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
