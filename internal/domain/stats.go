package domain

import "math"

// TargetStatus beschreibt das Ergebnis des Soll/Ist-Abgleichs.
type TargetStatus string

const (
	// TargetReached: das Glas reicht (deutlich) bis zum Zieldatum.
	TargetReached TargetStatus = "ja"
	// TargetTight: das Glas reicht nur knapp (innerhalb der Toleranz).
	TargetTight TargetStatus = "knapp"
	// TargetMissed: das Glas reicht voraussichtlich nicht bis zum Zieldatum.
	TargetMissed TargetStatus = "nein"
	// TargetUnknown: zu wenige Daten für eine Schätzung.
	TargetUnknown TargetStatus = "unbekannt"
)

// tightToleranceDays ist die Toleranz (in Tagen), innerhalb derer das Ergebnis
// als "knapp" gewertet wird.
const tightToleranceDays = 3

// ConsumptionPoint beschreibt den Verbrauch zwischen zwei aufeinanderfolgenden
// Messungen.
type ConsumptionPoint struct {
	// Date ist der Tag der späteren Messung.
	Date Date `json:"date"`
	// Net ist der Nutzinhalt zu diesem Zeitpunkt in Gramm.
	Net float64 `json:"net"`
	// Consumed ist die seit der vorherigen Messung verbrauchte Menge in Gramm.
	Consumed float64 `json:"consumed"`
	// Days ist die Anzahl Tage seit der vorherigen Messung.
	Days int `json:"days"`
	// PerDay ist der durchschnittliche Verbrauch pro Tag in diesem Intervall.
	PerDay float64 `json:"perDay"`
}

// Stats bündelt alle berechneten Kennzahlen und Schätzungen für ein Glas.
type Stats struct {
	JarID string `json:"jarId"`

	InitialNet float64 `json:"initialNet"` // anfänglicher Nutzinhalt (g)
	CurrentNet float64 `json:"currentNet"` // aktueller Nutzinhalt (g)

	TotalConsumed     float64 `json:"totalConsumed"`     // gesamt verbraucht (g)
	ConsumedSinceLast float64 `json:"consumedSinceLast"` // seit letzter Messung (g)

	DaysElapsed          int     `json:"daysElapsed"`          // Tage seit Start bis letzte Messung
	BurnRatePerDay       float64 `json:"burnRatePerDay"`       // Ø Verbrauch pro Kalendertag (g)
	BurnRatePerEatingDay float64 `json:"burnRatePerEatingDay"` // Ø Verbrauch pro Esstag (Sa/So) (g)

	PlannedDailyRate float64 `json:"plannedDailyRate"` // Soll-Verbrauch pro Tag (g)

	MaxConsumed float64 `json:"maxConsumed"` // größter Einzelverbrauch (g)
	MinConsumed float64 `json:"minConsumed"` // kleinster Einzelverbrauch (g)
	AvgConsumed float64 `json:"avgConsumed"` // Ø Verbrauch pro Messintervall (g)

	EstimatedEmptyDate *Date        `json:"estimatedEmptyDate"` // geschätztes Leerdatum
	TargetStatus       TargetStatus `json:"targetStatus"`       // Soll/Ist-Ergebnis
	TargetDiffDays     int          `json:"targetDiffDays"`     // Differenz Leerdatum − Zieldatum (Tage)

	Consumption []ConsumptionPoint `json:"consumption"` // Verbrauch je Messintervall
}

// countEatingDays zählt die Wochenend-Tage (Samstag/Sonntag) im Intervall
// [from, to] (beide inklusive).
func countEatingDays(from, to Date) int {
	if to.Before(from.Time) {
		return 0
	}
	count := 0
	for d := from; !d.After(to.Time); d = d.AddDays(1) {
		switch d.Weekday() {
		case 0, 6: // Sonntag (0), Samstag (6)
			count++
		}
	}
	return count
}

// ComputeStats berechnet alle Kennzahlen und Schätzungen für das Glas.
// today ist der Bezugstag für die Reichweiten-Schätzung.
func ComputeStats(j *Jar, today Date) Stats {
	s := Stats{
		JarID:        j.ID,
		InitialNet:   j.InitialNet(),
		CurrentNet:   j.InitialNet(),
		TargetStatus: TargetUnknown,
		Consumption:  []ConsumptionPoint{},
	}

	// Soll-Verbrauch pro Tag (idealer linearer Verbrauch bis zum Zieldatum).
	plannedDays := j.TargetDate.DaysSince(j.StartDate)
	if plannedDays > 0 {
		s.PlannedDailyRate = j.InitialNet() / float64(plannedDays)
	}

	ms := j.sortedMeasurements()
	if len(ms) == 0 {
		return s
	}

	last := ms[len(ms)-1]
	s.CurrentNet = last.GrossWeight - j.TareWeight
	s.TotalConsumed = s.InitialNet - s.CurrentNet

	// Verbrauch je Messintervall. Der erste Punkt vergleicht gegen den
	// Anfangszustand (Startdatum / voller Inhalt).
	prevDate := j.StartDate
	prevNet := s.InitialNet
	sumConsumed := 0.0
	s.MinConsumed = math.Inf(1)
	s.MaxConsumed = math.Inf(-1)

	for _, m := range ms {
		net := m.GrossWeight - j.TareWeight
		consumed := prevNet - net
		days := m.Date.DaysSince(prevDate)
		perDay := 0.0
		if days > 0 {
			perDay = consumed / float64(days)
		}
		s.Consumption = append(s.Consumption, ConsumptionPoint{
			Date:     m.Date,
			Net:      net,
			Consumed: consumed,
			Days:     days,
			PerDay:   perDay,
		})
		sumConsumed += consumed
		if consumed > s.MaxConsumed {
			s.MaxConsumed = consumed
		}
		if consumed < s.MinConsumed {
			s.MinConsumed = consumed
		}
		prevDate = m.Date
		prevNet = net
	}

	if len(s.Consumption) > 0 {
		s.ConsumedSinceLast = s.Consumption[len(s.Consumption)-1].Consumed
		s.AvgConsumed = sumConsumed / float64(len(s.Consumption))
	} else {
		s.MinConsumed = 0
		s.MaxConsumed = 0
	}

	// Burnrate pro Kalendertag und pro Esstag (Sa/So).
	s.DaysElapsed = last.Date.DaysSince(j.StartDate)
	if s.DaysElapsed > 0 {
		s.BurnRatePerDay = s.TotalConsumed / float64(s.DaysElapsed)
	}
	eatingDays := countEatingDays(j.StartDate, last.Date)
	if eatingDays > 0 {
		s.BurnRatePerEatingDay = s.TotalConsumed / float64(eatingDays)
	}

	// Reichweiten-Schätzung: bei aktueller Burnrate, ab heute.
	if s.BurnRatePerDay > 0 && s.CurrentNet > 0 {
		remainingDays := int(math.Ceil(s.CurrentNet / s.BurnRatePerDay))
		empty := today.AddDays(remainingDays)
		s.EstimatedEmptyDate = &empty

		s.TargetDiffDays = empty.DaysSince(j.TargetDate)
		switch {
		case s.TargetDiffDays >= tightToleranceDays:
			s.TargetStatus = TargetReached
		case s.TargetDiffDays <= -tightToleranceDays:
			s.TargetStatus = TargetMissed
		default:
			s.TargetStatus = TargetTight
		}
	} else if s.CurrentNet <= 0 {
		// Glas ist bereits leer.
		empty := last.Date
		s.EstimatedEmptyDate = &empty
		s.TargetDiffDays = empty.DaysSince(j.TargetDate)
		if s.TargetDiffDays >= 0 {
			s.TargetStatus = TargetReached
		} else {
			s.TargetStatus = TargetMissed
		}
	}

	return s
}
