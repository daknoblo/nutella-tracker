// Package demo erzeugt einen realistischen Beispiel-Datenbestand für die
// öffentliche Demo-Seite und die automatisierten Screenshots. Die Daten werden
// relativ zum Bezugstag aufgebaut, damit Zeitachsen und Prognosen immer aktuell
// wirken; bei gleichem Bezugstag ist das Ergebnis reproduzierbar.
package demo

import (
	"github.com/daknoblo/nutella-tracker/internal/domain"
	"github.com/daknoblo/nutella-tracker/internal/storage"
)

// Standardmaße eines 1-kg-Glases: Bruttofüllgewicht und Tara in Gramm.
const (
	grossFullWeight = 1200
	tareWeight      = 200
)

// ActiveJarID ist die ID des Glases, das in der Demo aktiv ist.
const ActiveJarID = "demo-sommer"

// jarSpec beschreibt ein Demo-Glas, bevor die Messungen erzeugt werden.
type jarSpec struct {
	id            string
	name          string
	startOffset   int       // Startdatum relativ zum Bezugstag (in Tagen)
	targetOffset  int       // Zieldatum relativ zum Bezugstag (in Tagen)
	lastOffset    int       // letzter Messtag relativ zum Bezugstag (in Tagen)
	consumption   []float64 // zyklisches Verbrauchsmuster je Esstag (g)
	emptyAtTheEnd bool      // letzte Messung auf "leer" setzen
}

// specs deckt bewusst alle Zustände ab, die die UI darstellen kann:
// planmäßig (grün), zu schneller Verbrauch (rot) und ein bereits leeres Glas.
var specs = []jarSpec{
	{
		id:           "demo-winter",
		name:         "Glas Winter",
		startOffset:  -140,
		targetOffset: -80,
		lastOffset:   -78,
		consumption:  []float64{58, 64, 51, 69, 55, 62},
		// Das Glas wurde bis zum Schluss aufgebraucht.
		emptyAtTheEnd: true,
	},
	{
		id:           "demo-sommer",
		name:         "Glas Sommer",
		startOffset:  -42,
		targetOffset: 18,
		lastOffset:   0,
		// Ø ca. 52 g pro Esstag – etwas sparsamer als geplant.
		consumption: []float64{48, 57, 44, 61, 50, 55, 46, 59},
	},
	{
		id:           "demo-urlaub",
		name:         "Glas Urlaub",
		startOffset:  -30,
		targetOffset: 30,
		lastOffset:   0,
		// Ø ca. 76 g pro Esstag – deutlich über Plan, das Glas reicht nicht.
		consumption: []float64{71, 84, 66, 92, 74, 80},
	},
}

// Jars liefert den Demo-Datenbestand relativ zum Bezugstag today.
func Jars(today domain.Date) []*domain.Jar {
	jars := make([]*domain.Jar, 0, len(specs))
	for _, spec := range specs {
		jars = append(jars, buildJar(spec, today))
	}
	return jars
}

// Seed schreibt den Demo-Datenbestand in den Datenspeicher.
func Seed(store *storage.Store, today domain.Date) error {
	for _, jar := range Jars(today) {
		if err := store.AddJar(jar); err != nil {
			return err
		}
	}
	return store.SetActive(ActiveJarID)
}

// buildJar erzeugt ein Glas samt Wochenend-Messungen (Sa/So) aus einer Spec.
func buildJar(spec jarSpec, today domain.Date) *domain.Jar {
	start := today.AddDays(spec.startOffset)
	last := today.AddDays(spec.lastOffset)

	jar := &domain.Jar{
		ID:              spec.id,
		Name:            spec.name,
		GrossFullWeight: grossFullWeight,
		TareWeight:      tareWeight,
		StartDate:       start,
		TargetDate:      today.AddDays(spec.targetOffset),
		Measurements:    []domain.Measurement{},
	}

	// Gegessen wird nur am Wochenende: jeder Sa/So nach dem Start ist ein Messtag.
	net := jar.InitialNet()
	step := 0
	for d := start.AddDays(1); !d.After(last.Time); d = d.AddDays(1) {
		if d.Weekday() != 0 && d.Weekday() != 6 {
			continue
		}
		net -= spec.consumption[step%len(spec.consumption)]
		step++
		if net < 0 {
			net = 0
		}
		jar.Measurements = append(jar.Measurements, domain.Measurement{
			Date:        d,
			GrossWeight: jar.TareWeight + net,
		})
	}

	if spec.emptyAtTheEnd && len(jar.Measurements) > 0 {
		jar.Measurements[len(jar.Measurements)-1].GrossWeight = jar.TareWeight
	}
	return jar
}
