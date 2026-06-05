// Command server startet den HTTP-Server des Nutella Trackers: er liefert die
// Web-UI aus und stellt die JSON-API bereit. Die Daten werden in einer
// einzelnen JSON-Datei gespeichert.
package main

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/daknoblo/nutella-tracker/internal/api"
	"github.com/daknoblo/nutella-tracker/internal/storage"
	"github.com/daknoblo/nutella-tracker/web"
)

func main() {
	port := envOr("PORT", "8080")
	dataFile := envOr("DATA_FILE", "data/nutella.json")

	store, err := storage.Open(dataFile)
	if err != nil {
		log.Fatalf("Datenspeicher konnte nicht geöffnet werden: %v", err)
	}

	mux := http.NewServeMux()

	// API-Routen registrieren.
	api.New(store).Routes(mux)

	// Statische Web-UI ausliefern (eingebettetes Dateisystem).
	staticFS, err := fs.Sub(web.Files, ".")
	if err != nil {
		log.Fatalf("Web-Assets konnten nicht geladen werden: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful Shutdown bei SIGINT/SIGTERM.
	go func() {
		log.Printf("Nutella Tracker läuft auf http://localhost:%s (Daten: %s)", port, dataFile)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server-Fehler: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Server wird beendet ...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Fehler beim Beenden: %v", err)
	}
}

// envOr liefert den Wert der Umgebungsvariablen key oder fallback.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// logRequests ist eine einfache Logging-Middleware.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	})
}
