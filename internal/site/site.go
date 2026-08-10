// Package site erzeugt die statische Projektseite für GitHub Pages: eine aus
// der README gerenderte Dokumentationsseite, eine lauffähige Demo der Web-UI
// mit Beispieldaten und die Metadaten für die automatisierten Screenshots.
package site

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	gmtext "github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"github.com/daknoblo/nutella-tracker/internal/api"
	"github.com/daknoblo/nutella-tracker/internal/demo"
	"github.com/daknoblo/nutella-tracker/internal/domain"
	"github.com/daknoblo/nutella-tracker/internal/storage"
)

//go:embed assets templates
var files embed.FS

// Options steuert die Generierung der Projektseite.
type Options struct {
	// OutDir ist das Zielverzeichnis (wird angelegt).
	OutDir string
	// WebDir enthält die Quellen der Web-UI (index.html, app.js, style.css).
	WebDir string
	// Readme ist der Pfad zur README.md, aus der die Doku gerendert wird.
	Readme string
	// RepoURL ist die Repository-Basis-URL für relative Links in der README.
	RepoURL string
	// Branch ist der Branch, auf den relative README-Links zeigen.
	Branch string
	// Today ist der Bezugstag der Demo-Daten.
	Today domain.Date
}

// webAssets sind die Dateien der Web-UI, die in die Demo kopiert werden.
var webAssets = []string{"app.js", "style.css"}

// Generate schreibt die komplette Seite nach opts.OutDir.
func Generate(opts Options) error {
	if opts.OutDir == "" {
		return fmt.Errorf("Zielverzeichnis fehlt")
	}
	if opts.Branch == "" {
		opts.Branch = "main"
	}
	if opts.Today.IsZero() {
		opts.Today = domain.Today()
	}

	for _, dir := range []string{
		filepath.Join(opts.OutDir, "assets", "screenshots"),
		filepath.Join(opts.OutDir, "demo", "api"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("Verzeichnis anlegen: %w", err)
		}
	}

	// Verhindert, dass GitHub Pages die Dateien durch Jekyll schickt.
	if err := os.WriteFile(filepath.Join(opts.OutDir, ".nojekyll"), nil, 0o644); err != nil {
		return fmt.Errorf(".nojekyll schreiben: %w", err)
	}

	if err := writeDemo(opts); err != nil {
		return err
	}
	if err := writeShots(opts.OutDir); err != nil {
		return err
	}
	return writeIndex(opts)
}

// writeDemo legt die lauffähige Demo der Web-UI inkl. API-Snapshot ab.
func writeDemo(opts Options) error {
	demoDir := filepath.Join(opts.OutDir, "demo")

	snapshot, err := jarsSnapshot(opts.Today)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(demoDir, "api", "jars.json"), snapshot, 0o644); err != nil {
		return fmt.Errorf("Demo-Daten schreiben: %w", err)
	}

	for _, name := range webAssets {
		content, err := os.ReadFile(filepath.Join(opts.WebDir, name))
		if err != nil {
			return fmt.Errorf("Web-Asset %s lesen: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(demoDir, name), content, 0o644); err != nil {
			return fmt.Errorf("Web-Asset %s schreiben: %w", name, err)
		}
	}

	if err := copyEmbedded("assets/demo-api.js", filepath.Join(demoDir, "demo-api.js")); err != nil {
		return err
	}
	if err := copyEmbedded("assets/site.css", filepath.Join(opts.OutDir, "assets", "site.css")); err != nil {
		return err
	}

	source, err := os.ReadFile(filepath.Join(opts.WebDir, "index.html"))
	if err != nil {
		return fmt.Errorf("index.html der Web-UI lesen: %w", err)
	}
	banner, err := files.ReadFile("assets/demo-banner.html")
	if err != nil {
		return fmt.Errorf("Demo-Banner lesen: %w", err)
	}
	page, err := patchDemoHTML(string(source), string(banner))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(demoDir, "index.html"), []byte(page), 0o644); err != nil {
		return fmt.Errorf("Demo-Seite schreiben: %w", err)
	}
	return nil
}

// jarsSnapshot erzeugt die Demo-Daten und ruft damit die echte API auf, damit
// der Snapshot exakt dem Format von GET /api/jars entspricht.
func jarsSnapshot(today domain.Date) ([]byte, error) {
	dir, err := os.MkdirTemp("", "nutella-demo-*")
	if err != nil {
		return nil, fmt.Errorf("temporäres Verzeichnis anlegen: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	store, err := storage.Open(filepath.Join(dir, "demo.json"))
	if err != nil {
		return nil, fmt.Errorf("Demo-Datenspeicher öffnen: %w", err)
	}
	if err := demo.Seed(store, today); err != nil {
		return nil, fmt.Errorf("Demo-Daten erzeugen: %w", err)
	}

	mux := http.NewServeMux()
	api.New(store, nil).Routes(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/jars", nil))
	if rec.Code != http.StatusOK {
		return nil, fmt.Errorf("GET /api/jars lieferte Status %d", rec.Code)
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, rec.Body.Bytes(), "", "  "); err != nil {
		return nil, fmt.Errorf("Snapshot formatieren: %w", err)
	}
	return pretty.Bytes(), nil
}

// patchDemoHTML baut die Web-UI zur statischen Demo um: Hinweis-Banner einfügen
// und die API durch den Demo-Shim ersetzen.
func patchDemoHTML(page, banner string) (string, error) {
	replacements := []struct{ old, new string }{
		{"<title>Nutella Tracker</title>", "<title>Nutella Tracker – Demo</title>"},
		{"  <main>\n", banner + "\n  <main>\n"},
		{`  <script src="app.js"></script>`, "  <script src=\"demo-api.js\"></script>\n  <script src=\"app.js\"></script>"},
	}
	for _, r := range replacements {
		if !strings.Contains(page, r.old) {
			return "", fmt.Errorf("Demo-Umbau fehlgeschlagen: %q nicht in web/index.html gefunden", r.old)
		}
		page = strings.Replace(page, r.old, r.new, 1)
	}
	return page, nil
}

// writeShots legt die Screenshot-Definitionen für den Screenshot-Runner ab.
func writeShots(outDir string) error {
	data, err := json.MarshalIndent(Shots(), "", "  ")
	if err != nil {
		return fmt.Errorf("Screenshot-Liste serialisieren: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "screenshots.json"), data, 0o644); err != nil {
		return fmt.Errorf("Screenshot-Liste schreiben: %w", err)
	}
	return nil
}

// pageData sind die Daten des Seiten-Templates.
type pageData struct {
	Title       string
	Description string
	RepoURL     string
	Readme      template.HTML
	Shots       []Shot
	Generated   string
}

// writeIndex rendert die README zur Projektseite.
func writeIndex(opts Options) error {
	source, err := os.ReadFile(opts.Readme)
	if err != nil {
		return fmt.Errorf("README lesen: %w", err)
	}

	title, description, body, err := renderReadme(source, opts.RepoURL, opts.Branch)
	if err != nil {
		return err
	}

	tmpl, err := template.ParseFS(files, "templates/index.html.tmpl")
	if err != nil {
		return fmt.Errorf("Template laden: %w", err)
	}

	var out bytes.Buffer
	err = tmpl.Execute(&out, pageData{
		Title:       title,
		Description: description,
		RepoURL:     opts.RepoURL,
		Readme:      template.HTML(body), //nolint:gosec // gerenderte README aus dem eigenen Repository
		Shots:       Shots(),
		Generated:   time.Now().Format("2006-01-02"),
	})
	if err != nil {
		return fmt.Errorf("Seite rendern: %w", err)
	}
	if err := os.WriteFile(filepath.Join(opts.OutDir, "index.html"), out.Bytes(), 0o644); err != nil {
		return fmt.Errorf("Seite schreiben: %w", err)
	}
	return nil
}

// renderReadme rendert die README nach HTML und liefert zusätzlich Titel und
// Kurzbeschreibung. Die Überschrift der ersten Ebene entfällt, weil der Titel
// bereits im Seitenkopf steht.
func renderReadme(source []byte, repoURL, branch string) (title, description, body string, err error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithASTTransformers(util.Prioritized(&readmeTransformer{
				repoURL: strings.TrimSuffix(repoURL, "/"),
				branch:  branch,
			}, 100)),
		),
		goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
	)

	doc := md.Parser().Parse(gmtext.NewReader(source))
	// Titel und Einleitung stammen aus der README, bevor die Titel-Überschrift
	// entfernt wird (sie steht auf der Seite bereits im Kopf).
	title = cleanTitle(firstHeading(doc, source))
	description = firstPlainParagraph(doc, source)
	removeFirstHeading(doc)

	var out bytes.Buffer
	if err := md.Renderer().Render(&out, source, doc); err != nil {
		return "", "", "", fmt.Errorf("README rendern: %w", err)
	}
	return title, description, out.String(), nil
}

// readmeTransformer macht die README seitentauglich: relative Links zeigen auf
// das Repository.
type readmeTransformer struct {
	repoURL string
	branch  string
}

func (t *readmeTransformer) Transform(doc *ast.Document, _ gmtext.Reader, _ parser.Context) {
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.Link:
			v.Destination = []byte(t.absolute(string(v.Destination), "blob"))
		case *ast.Image:
			v.Destination = []byte(t.absolute(string(v.Destination), "raw"))
		}
		return ast.WalkContinue, nil
	})
}

// absolute wandelt einen repo-relativen Pfad in eine GitHub-URL um.
func (t *readmeTransformer) absolute(dest, kind string) string {
	if dest == "" || strings.Contains(dest, "://") ||
		strings.HasPrefix(dest, "#") || strings.HasPrefix(dest, "//") ||
		strings.HasPrefix(dest, "mailto:") {
		return dest
	}
	return fmt.Sprintf("%s/%s/%s/%s", t.repoURL, kind, t.branch, strings.TrimPrefix(dest, "./"))
}

// firstHeading liefert den Text der ersten Überschrift (Seitentitel).
func firstHeading(doc ast.Node, source []byte) string {
	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		if h, ok := n.(*ast.Heading); ok {
			return strings.TrimSpace(string(h.Text(source))) //nolint:staticcheck // Text ist für Überschriften ausreichend
		}
	}
	return "Nutella Tracker"
}

// removeFirstHeading entfernt die Titel-Überschrift der README.
func removeFirstHeading(doc ast.Node) {
	if h, ok := doc.FirstChild().(*ast.Heading); ok && h.Level == 1 {
		doc.RemoveChild(doc, h)
	}
}

// cleanTitle entfernt führende Symbole (z. B. Emojis) aus der Überschrift, weil
// die Seite bereits ein eigenes Logo trägt.
func cleanTitle(title string) string {
	trimmed := strings.TrimLeftFunc(title, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if trimmed == "" {
		return title
	}
	return trimmed
}

// firstPlainParagraph liefert den ersten Absatz ohne Links/Bilder – das ist der
// Einleitungssatz der README (die Badge-Zeile wird so übersprungen).
func firstPlainParagraph(doc ast.Node, source []byte) string {
	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		p, ok := n.(*ast.Paragraph)
		if !ok || containsLink(p) {
			continue
		}
		text := strings.Join(strings.Fields(string(p.Text(source))), " ") //nolint:staticcheck // reiner Fließtext
		if text != "" {
			return text
		}
	}
	return ""
}

// containsLink meldet, ob ein Knoten Links oder Bilder enthält.
func containsLink(n ast.Node) bool {
	found := false
	_ = ast.Walk(n, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch child.(type) {
		case *ast.Link, *ast.Image, *ast.AutoLink:
			found = true
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	return found
}

// copyEmbedded kopiert eine eingebettete Datei ins Zielverzeichnis.
func copyEmbedded(name, dest string) error {
	content, err := files.ReadFile(name)
	if err != nil {
		return fmt.Errorf("eingebettete Datei %s lesen: %w", name, err)
	}
	if err := os.WriteFile(dest, content, 0o644); err != nil {
		return fmt.Errorf("%s schreiben: %w", dest, err)
	}
	return nil
}
