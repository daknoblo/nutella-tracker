package demo

import (
	"testing"

	"github.com/daknoblo/nutella-tracker/internal/domain"
)

func TestJarsAreValidAndCoverAllTargetStates(t *testing.T) {
	today := domain.NewDate(2026, 8, 10)
	jars := Jars(today)

	if len(jars) != len(specs) {
		t.Fatalf("erwartet %d Gläser, erhalten %d", len(specs), len(jars))
	}

	seen := map[domain.TargetStatus]bool{}
	for _, jar := range jars {
		if err := jar.Validate(); err != nil {
			t.Errorf("Glas %s ist ungültig: %v", jar.ID, err)
		}
		if len(jar.Measurements) < 3 {
			t.Errorf("Glas %s hat zu wenige Messungen (%d)", jar.ID, len(jar.Measurements))
		}
		for _, m := range jar.Measurements {
			if m.GrossWeight < jar.TareWeight || m.GrossWeight > jar.GrossFullWeight {
				t.Errorf("Glas %s: Messung %s außerhalb des gültigen Bereichs (%.1f g)",
					jar.ID, m.Date.Format("2006-01-02"), m.GrossWeight)
			}
		}
		seen[domain.ComputeStats(jar, today).TargetStatus] = true
	}

	// Die Demo soll sowohl den grünen als auch den roten Status zeigen.
	for _, want := range []domain.TargetStatus{domain.TargetReached, domain.TargetMissed} {
		if !seen[want] {
			t.Errorf("kein Demo-Glas mit Status %q", want)
		}
	}
}

func TestMeasurementsOnlyOnWeekends(t *testing.T) {
	for _, jar := range Jars(domain.NewDate(2026, 8, 10)) {
		for _, m := range jar.Measurements {
			if wd := m.Date.Weekday(); wd != 0 && wd != 6 {
				t.Errorf("Glas %s: Messung am %s ist kein Wochenendtag", jar.ID, wd)
			}
		}
	}
}
