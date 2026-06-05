// Package storage kapselt die Persistenz der Gläser/Messungen in einer
// einzelnen JSON-Datei. Schreibvorgänge erfolgen atomar (temporäre Datei +
// Rename), um Korruption zu vermeiden.
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/daknoblo/nutella-tracker/internal/domain"
)

// ErrNotFound wird zurückgegeben, wenn ein Glas nicht existiert.
var ErrNotFound = errors.New("Glas nicht gefunden")

// database ist die serialisierte Struktur der JSON-Datei.
type database struct {
	Jars        []*domain.Jar `json:"jars"`
	ActiveJarID string        `json:"activeJarId"`
}

// Store hält die Daten im Speicher und synchronisiert den Zugriff.
type Store struct {
	path string
	mu   sync.RWMutex
	data database
}

// Open lädt den Datenspeicher aus path. Existiert die Datei nicht, wird ein
// leerer Speicher angelegt (Verzeichnis wird bei Bedarf erstellt).
func Open(path string) (*Store, error) {
	s := &Store{path: path}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("Datenverzeichnis anlegen: %w", err)
	}

	b, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		// Frischer Start: leere Datei schreiben.
		if err := s.save(); err != nil {
			return nil, err
		}
	case err != nil:
		return nil, fmt.Errorf("Datendatei lesen: %w", err)
	default:
		if len(b) > 0 {
			if err := json.Unmarshal(b, &s.data); err != nil {
				return nil, fmt.Errorf("Datendatei parsen: %w", err)
			}
		}
	}
	return s, nil
}

// save schreibt den aktuellen Zustand atomar in die JSON-Datei.
// Aufrufer müssen den Schreib-Lock halten.
func (s *Store) save() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("Daten serialisieren: %w", err)
	}

	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, ".nutella-*.tmp")
	if err != nil {
		return fmt.Errorf("temporäre Datei anlegen: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // best effort, falls Rename fehlschlägt

	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return fmt.Errorf("temporäre Datei schreiben: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("temporäre Datei syncen: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("temporäre Datei schließen: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("Datei ersetzen: %w", err)
	}
	return nil
}

// ListJars liefert eine Kopie aller Gläser.
func (s *Store) ListJars() []*domain.Jar {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*domain.Jar, len(s.data.Jars))
	copy(out, s.data.Jars)
	return out
}

// GetJar liefert das Glas mit der angegebenen ID.
func (s *Store) GetJar(id string) (*domain.Jar, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, j := range s.data.Jars {
		if j.ID == id {
			return j, nil
		}
	}
	return nil, ErrNotFound
}

// ActiveJarID liefert die ID des aktiven Glases (leer, wenn keines aktiv ist).
func (s *Store) ActiveJarID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.ActiveJarID
}

// ActiveJar liefert das aktive Glas oder ErrNotFound.
func (s *Store) ActiveJar() (*domain.Jar, error) {
	s.mu.RLock()
	id := s.data.ActiveJarID
	s.mu.RUnlock()
	if id == "" {
		return nil, ErrNotFound
	}
	return s.GetJar(id)
}

// AddJar fügt ein neues Glas hinzu und macht es zum aktiven Glas.
func (s *Store) AddJar(j *domain.Jar) error {
	if err := j.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.data.Jars {
		if existing.ID == j.ID {
			return fmt.Errorf("Glas-ID %q existiert bereits", j.ID)
		}
	}
	if j.Measurements == nil {
		j.Measurements = []domain.Measurement{}
	}
	s.data.Jars = append(s.data.Jars, j)
	s.data.ActiveJarID = j.ID
	return s.save()
}

// UpdateJar ersetzt die Stammdaten eines Glases (Messungen bleiben erhalten).
func (s *Store) UpdateJar(j *domain.Jar) error {
	if err := j.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Jars {
		if existing.ID == j.ID {
			// Messungen des Bestands beibehalten.
			j.Measurements = existing.Measurements
			s.data.Jars[i] = j
			return s.save()
		}
	}
	return ErrNotFound
}

// DeleteJar entfernt ein Glas. War es aktiv, wird das aktive Glas zurückgesetzt.
func (s *Store) DeleteJar(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Jars {
		if existing.ID == id {
			s.data.Jars = append(s.data.Jars[:i], s.data.Jars[i+1:]...)
			if s.data.ActiveJarID == id {
				s.data.ActiveJarID = ""
				if len(s.data.Jars) > 0 {
					s.data.ActiveJarID = s.data.Jars[len(s.data.Jars)-1].ID
				}
			}
			return s.save()
		}
	}
	return ErrNotFound
}

// SetActive setzt das aktive Glas.
func (s *Store) SetActive(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.data.Jars {
		if existing.ID == id {
			s.data.ActiveJarID = id
			return s.save()
		}
	}
	return ErrNotFound
}

// AddMeasurement fügt einem Glas eine Messung hinzu.
func (s *Store) AddMeasurement(jarID string, m domain.Measurement) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range s.data.Jars {
		if j.ID == jarID {
			j.Measurements = append(j.Measurements, m)
			return s.save()
		}
	}
	return ErrNotFound
}

// DeleteMeasurement entfernt die Messung am angegebenen Index.
func (s *Store) DeleteMeasurement(jarID string, index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range s.data.Jars {
		if j.ID == jarID {
			if index < 0 || index >= len(j.Measurements) {
				return errors.New("ungültiger Messungs-Index")
			}
			j.Measurements = append(j.Measurements[:index], j.Measurements[index+1:]...)
			return s.save()
		}
	}
	return ErrNotFound
}
