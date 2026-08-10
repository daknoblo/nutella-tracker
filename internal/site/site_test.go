package site

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	out := t.TempDir()
	repoRoot := filepath.Join("..", "..")

	err := Generate(Options{
		OutDir:  out,
		WebDir:  filepath.Join(repoRoot, "web"),
		Readme:  filepath.Join(repoRoot, "README.md"),
		RepoURL: "https://github.com/daknoblo/nutella-tracker",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	for _, name := range []string{
		".nojekyll",
		"index.html",
		"screenshots.json",
		filepath.Join("assets", "site.css"),
		filepath.Join("demo", "index.html"),
		filepath.Join("demo", "app.js"),
		filepath.Join("demo", "style.css"),
		filepath.Join("demo", "demo-api.js"),
		filepath.Join("demo", "api", "jars.json"),
	} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Errorf("erwartete Datei fehlt: %s (%v)", name, err)
		}
	}

	index := readFile(t, filepath.Join(out, "index.html"))
	if !strings.Contains(index, `href="demo/"`) {
		t.Error("Projektseite verlinkt die Demo nicht")
	}
	if !strings.Contains(index, "<title>nutella-tracker</title>") {
		t.Error("Projektseite übernimmt den Titel der README nicht")
	}
	for _, shot := range Shots() {
		if !strings.Contains(index, "assets/screenshots/"+shot.Name+".png") {
			t.Errorf("Screenshot %s fehlt in der Galerie", shot.Name)
		}
	}

	demoPage := readFile(t, filepath.Join(out, "demo", "index.html"))
	if !strings.Contains(demoPage, `<script src="demo-api.js"></script>`) {
		t.Error("Demo-Seite bindet den API-Shim nicht ein")
	}

	var views []struct {
		Jar struct {
			ID           string `json:"id"`
			Measurements []struct {
				GrossWeight float64 `json:"grossWeight"`
			} `json:"measurements"`
		} `json:"jar"`
		Stats  map[string]any `json:"stats"`
		Active bool           `json:"active"`
	}
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(out, "demo", "api", "jars.json"))), &views); err != nil {
		t.Fatalf("Demo-Daten sind kein gültiges JSON: %v", err)
	}
	if len(views) == 0 {
		t.Fatal("Demo-Daten enthalten keine Gläser")
	}
	for _, v := range views {
		if len(v.Jar.Measurements) == 0 {
			t.Errorf("Glas %s hat keine Messungen", v.Jar.ID)
		}
		if len(v.Stats) == 0 {
			t.Errorf("Glas %s hat keine Statistik", v.Jar.ID)
		}
	}
}

func TestAbsoluteKeepsExternalLinks(t *testing.T) {
	tr := &readmeTransformer{repoURL: "https://example.com/repo", branch: "main"}
	cases := map[string]string{
		"https://example.org/x": "https://example.org/x",
		"#anchor":               "#anchor",
		"LICENSE":               "https://example.com/repo/blob/main/LICENSE",
		"./go.mod":              "https://example.com/repo/blob/main/go.mod",
	}
	for in, want := range cases {
		if got := tr.absolute(in, "blob"); got != want {
			t.Errorf("absolute(%q) = %q, erwartet %q", in, got, want)
		}
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path) //nolint:gosec // Testpfad aus t.TempDir()
	if err != nil {
		t.Fatalf("%s lesen: %v", path, err)
	}
	return string(content)
}
