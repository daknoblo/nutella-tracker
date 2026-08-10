// Command gensite erzeugt die statische Projektseite (GitHub Pages): die aus der
// README gerenderte Dokumentation, eine lauffähige Demo der Web-UI mit
// Beispieldaten und die Screenshot-Definitionen.
package main

import (
	"flag"
	"log"

	"github.com/daknoblo/nutella-tracker/internal/site"
)

func main() {
	opts := site.Options{}
	flag.StringVar(&opts.OutDir, "out", "dist", "Zielverzeichnis der generierten Seite")
	flag.StringVar(&opts.WebDir, "web", "web", "Verzeichnis der Web-UI")
	flag.StringVar(&opts.Readme, "readme", "README.md", "Pfad zur README")
	flag.StringVar(&opts.RepoURL, "repo", "https://github.com/daknoblo/nutella-tracker", "Repository-URL für relative Links")
	flag.StringVar(&opts.Branch, "branch", "main", "Branch für relative Links")
	flag.Parse()

	if err := site.Generate(opts); err != nil {
		log.Fatalf("Seite konnte nicht erzeugt werden: %v", err)
	}
	log.Printf("Seite erzeugt in %s", opts.OutDir)
}
