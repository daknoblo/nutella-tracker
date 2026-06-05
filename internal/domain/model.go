// Package domain enthält das Datenmodell und die fachliche Berechnungslogik
// für den Nutella Tracker (Gläser, Messungen, Statistiken, Schätzungen).
package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// dateLayout ist das Format, in dem Datumswerte als JSON gespeichert werden.
const dateLayout = "2006-01-02"

// Date ist ein reiner Kalendertag (ohne Uhrzeit/Zeitzone), der in JSON als
// "YYYY-MM-DD" serialisiert wird.
type Date struct {
	time.Time
}

// NewDate erzeugt ein Date aus Jahr/Monat/Tag (in UTC, ohne Uhrzeit).
func NewDate(year int, month time.Month, day int) Date {
	return Date{time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

// DateOf normalisiert einen beliebigen Zeitpunkt auf seinen Kalendertag.
func DateOf(t time.Time) Date {
	return NewDate(t.Year(), t.Month(), t.Day())
}

// Today liefert den heutigen Kalendertag (lokale Zeit).
func Today() Date {
	return DateOf(time.Now())
}

// MarshalJSON serialisiert das Date als "YYYY-MM-DD".
func (d Date) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.Format(dateLayout) + `"`), nil
}

// UnmarshalJSON liest ein Date aus "YYYY-MM-DD".
func (d *Date) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		return nil
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return fmt.Errorf("ungültiges Datum %q: %w", s, err)
	}
	d.Time = t
	return nil
}

// DaysSince liefert die Anzahl voller Kalendertage von other bis d.
func (d Date) DaysSince(other Date) int {
	return int(d.Sub(other.Time).Hours() / 24)
}

// AddDays gibt ein neues Date n Tage in der Zukunft (oder Vergangenheit) zurück.
func (d Date) AddDays(n int) Date {
	return DateOf(d.AddDate(0, 0, n))
}

// Measurement ist eine einzelne Wiege-Messung (Weigh-In) eines Glases.
type Measurement struct {
	// Date ist der Tag der Messung.
	Date Date `json:"date"`
	// GrossWeight ist das aktuelle Bruttogewicht (Glas + Restinhalt) in Gramm.
	GrossWeight float64 `json:"grossWeight"`
}

// Jar ist ein einzelnes Nutella-Glas mit seiner Mess-Historie.
type Jar struct {
	// ID ist die eindeutige Kennung des Glases.
	ID string `json:"id"`
	// Name ist eine optionale Bezeichnung.
	Name string `json:"name"`
	// GrossFullWeight ist das Bruttofüllgewicht (volles Glas inkl. Glas) in Gramm.
	GrossFullWeight float64 `json:"grossFullWeight"`
	// TareWeight ist das Leergewicht des Glases (Tara) in Gramm.
	TareWeight float64 `json:"tareWeight"`
	// StartDate ist das Startdatum (Beginn des Trackings).
	StartDate Date `json:"startDate"`
	// TargetDate ist das geplante Zieldatum, bis zu dem das Glas reichen soll.
	TargetDate Date `json:"targetDate"`
	// Measurements ist die chronologische Liste der Messungen.
	Measurements []Measurement `json:"measurements"`
}

// Validate prüft die fachliche Konsistenz eines Glases.
func (j *Jar) Validate() error {
	if strings.TrimSpace(j.ID) == "" {
		return errors.New("Glas-ID darf nicht leer sein")
	}
	if j.GrossFullWeight <= 0 {
		return errors.New("Bruttofüllgewicht muss größer als 0 sein")
	}
	if j.TareWeight < 0 {
		return errors.New("Leergewicht (Tara) darf nicht negativ sein")
	}
	if j.TareWeight >= j.GrossFullWeight {
		return errors.New("Leergewicht muss kleiner als das Bruttofüllgewicht sein")
	}
	if j.TargetDate.Before(j.StartDate.Time) {
		return errors.New("Zieldatum darf nicht vor dem Startdatum liegen")
	}
	return nil
}

// InitialNet liefert den anfänglichen Nutzinhalt (Füllgewicht abzüglich Tara) in Gramm.
func (j *Jar) InitialNet() float64 {
	return j.GrossFullWeight - j.TareWeight
}

// sortedMeasurements gibt die Messungen chronologisch sortiert zurück (Kopie).
func (j *Jar) sortedMeasurements() []Measurement {
	ms := make([]Measurement, len(j.Measurements))
	copy(ms, j.Measurements)
	// einfache stabile Sortierung nach Datum
	for i := 1; i < len(ms); i++ {
		for k := i; k > 0 && ms[k].Date.Before(ms[k-1].Date.Time); k-- {
			ms[k], ms[k-1] = ms[k-1], ms[k]
		}
	}
	return ms
}
