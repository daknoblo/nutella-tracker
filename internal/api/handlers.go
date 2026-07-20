// Package api stellt die JSON-HTTP-Schnittstelle des Nutella Trackers bereit.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/daknoblo/nutella-tracker/internal/domain"
	"github.com/daknoblo/nutella-tracker/internal/storage"
	"github.com/daknoblo/nutella-tracker/internal/vision"
)

// maxUploadBytes begrenzt die Größe hochgeladener Fotos (10 MB).
const maxUploadBytes = 10 << 20

// Server bündelt den Datenspeicher und stellt die API-Handler bereit.
type Server struct {
	store  *storage.Store
	vision *vision.Client
}

// New erzeugt einen neuen API-Server. visionClient darf nil sein, wenn keine
// Foto-Erkennung konfiguriert ist.
func New(store *storage.Store, visionClient *vision.Client) *Server {
	return &Server{store: store, vision: visionClient}
}

// Routes registriert alle API-Routen am übergebenen Mux.
func (s *Server) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/jars", s.handleListJars)
	mux.HandleFunc("POST /api/jars", s.handleCreateJar)
	mux.HandleFunc("GET /api/jars/{id}", s.handleGetJar)
	mux.HandleFunc("PUT /api/jars/{id}", s.handleUpdateJar)
	mux.HandleFunc("DELETE /api/jars/{id}", s.handleDeleteJar)
	mux.HandleFunc("POST /api/jars/{id}/activate", s.handleActivateJar)
	mux.HandleFunc("GET /api/jars/{id}/stats", s.handleJarStats)
	mux.HandleFunc("POST /api/jars/{id}/measurements", s.handleAddMeasurement)
	mux.HandleFunc("DELETE /api/jars/{id}/measurements/{index}", s.handleDeleteMeasurement)
	mux.HandleFunc("GET /api/active", s.handleActive)
	mux.HandleFunc("GET /api/config", s.handleConfig)
	mux.HandleFunc("POST /api/vision/recognize", s.handleVisionRecognize)
}

// jarView ist die API-Repräsentation eines Glases inkl. Statistik.
type jarView struct {
	Jar    *domain.Jar  `json:"jar"`
	Stats  domain.Stats `json:"stats"`
	Active bool         `json:"active"`
}

func (s *Server) handleListJars(w http.ResponseWriter, r *http.Request) {
	jars := s.store.ListJars()
	activeID := s.store.ActiveJarID()
	today := domain.Today()

	views := make([]jarView, 0, len(jars))
	for _, j := range jars {
		views = append(views, jarView{
			Jar:    j,
			Stats:  domain.ComputeStats(j, today),
			Active: j.ID == activeID,
		})
	}
	writeJSON(w, http.StatusOK, views)
}

// jarInput sind die vom Client gelieferten Stammdaten eines Glases.
type jarInput struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	GrossFullWeight float64     `json:"grossFullWeight"`
	TareWeight      float64     `json:"tareWeight"`
	StartDate       domain.Date `json:"startDate"`
	TargetDate      domain.Date `json:"targetDate"`
}

func (s *Server) handleCreateJar(w http.ResponseWriter, r *http.Request) {
	var in jarInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if in.ID == "" {
		in.ID = fmt.Sprintf("jar-%d", time.Now().UnixNano())
	}
	jar := &domain.Jar{
		ID:              in.ID,
		Name:            in.Name,
		GrossFullWeight: in.GrossFullWeight,
		TareWeight:      in.TareWeight,
		StartDate:       in.StartDate,
		TargetDate:      in.TargetDate,
		Measurements:    []domain.Measurement{},
	}
	if err := s.store.AddJar(jar); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, s.viewFor(jar))
}

func (s *Server) handleGetJar(w http.ResponseWriter, r *http.Request) {
	jar, err := s.store.GetJar(r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.viewFor(jar))
}

func (s *Server) handleUpdateJar(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in jarInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	jar := &domain.Jar{
		ID:              id,
		Name:            in.Name,
		GrossFullWeight: in.GrossFullWeight,
		TareWeight:      in.TareWeight,
		StartDate:       in.StartDate,
		TargetDate:      in.TargetDate,
	}
	if err := s.store.UpdateJar(jar); err != nil {
		writeStoreError(w, err)
		return
	}
	updated, _ := s.store.GetJar(id)
	writeJSON(w, http.StatusOK, s.viewFor(updated))
}

func (s *Server) handleDeleteJar(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteJar(r.PathValue("id")); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleActivateJar(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.SetActive(id); err != nil {
		writeStoreError(w, err)
		return
	}
	jar, _ := s.store.GetJar(id)
	writeJSON(w, http.StatusOK, s.viewFor(jar))
}

func (s *Server) handleJarStats(w http.ResponseWriter, r *http.Request) {
	jar, err := s.store.GetJar(r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, domain.ComputeStats(jar, domain.Today()))
}

// measurementInput ist die vom Client gelieferte Messung.
type measurementInput struct {
	Date        *domain.Date `json:"date"`
	GrossWeight float64      `json:"grossWeight"`
}

func (s *Server) handleAddMeasurement(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in measurementInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if in.GrossWeight <= 0 {
		writeError(w, http.StatusBadRequest, errors.New("Bruttogewicht muss größer als 0 sein"))
		return
	}
	date := domain.Today()
	if in.Date != nil {
		date = *in.Date
	}
	m := domain.Measurement{Date: date, GrossWeight: in.GrossWeight}
	if err := s.store.AddMeasurement(id, m); err != nil {
		writeStoreError(w, err)
		return
	}
	jar, _ := s.store.GetJar(id)
	writeJSON(w, http.StatusCreated, s.viewFor(jar))
}

func (s *Server) handleDeleteMeasurement(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("ungültiger Index"))
		return
	}
	if err := s.store.DeleteMeasurement(id, index); err != nil {
		writeStoreError(w, err)
		return
	}
	jar, _ := s.store.GetJar(id)
	writeJSON(w, http.StatusOK, s.viewFor(jar))
}

func (s *Server) handleActive(w http.ResponseWriter, r *http.Request) {
	jar, err := s.store.ActiveJar()
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.viewFor(jar))
}

// viewFor baut die API-Repräsentation eines Glases.
func (s *Server) viewFor(j *domain.Jar) jarView {
	return jarView{
		Jar:    j,
		Stats:  domain.ComputeStats(j, domain.Today()),
		Active: j.ID == s.store.ActiveJarID(),
	}
}

// handleConfig meldet der UI, welche optionalen Funktionen verfügbar sind.
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{
		"visionEnabled": s.vision != nil && s.vision.Enabled(),
	})
}

// handleVisionRecognize nimmt ein Foto entgegen, lässt den Gewichtswert vom
// Vision-Modell auslesen und gibt ihn zurück. Das Bild wird nicht gespeichert.
func (s *Server) handleVisionRecognize(w http.ResponseWriter, r *http.Request) {
	if s.vision == nil || !s.vision.Enabled() {
		writeError(w, http.StatusServiceUnavailable, errors.New("Foto-Erkennung ist nicht konfiguriert"))
		return
	}

	// Upload-Größe begrenzen.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

	file, header, err := r.FormFile("photo")
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("kein Foto im Feld 'photo' gefunden"))
		return
	}
	defer file.Close()

	image, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("Foto konnte nicht gelesen werden: %w", err))
		return
	}

	contentType := header.Header.Get("Content-Type")

	grams, err := s.vision.RecognizeWeight(r.Context(), image, contentType)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]float64{"grossWeight": grams})
}

// --- Hilfsfunktionen ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeError(w, http.StatusBadRequest, err)
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("ungültige Anfrage: %w", err)
	}
	return nil
}
