package domain

import (
	"math"
	"testing"
	"time"
)

// floatEquals vergleicht zwei Floats mit kleiner Toleranz.
func floatEquals(a, b float64) bool {
	return math.Abs(a-b) < 1e-6
}

func TestInitialNet(t *testing.T) {
	j := &Jar{GrossFullWeight: 1200, TareWeight: 200}
	if got := j.InitialNet(); !floatEquals(got, 1000) {
		t.Fatalf("InitialNet = %v, erwartet 1000", got)
	}
}

func TestDateDaysSinceAndAddDays(t *testing.T) {
	start := NewDate(2026, time.January, 1)
	end := NewDate(2026, time.January, 11)
	if d := end.DaysSince(start); d != 10 {
		t.Fatalf("DaysSince = %d, erwartet 10", d)
	}
	if got := start.AddDays(10); !got.Equal(end.Time) {
		t.Fatalf("AddDays = %v, erwartet %v", got, end)
	}
}

func TestCountEatingDays(t *testing.T) {
	// Sa 2026-01-03, So 2026-01-04, Sa 2026-01-10, So 2026-01-11
	from := NewDate(2026, time.January, 1) // Donnerstag
	to := NewDate(2026, time.January, 11)  // Sonntag
	if got := countEatingDays(from, to); got != 4 {
		t.Fatalf("countEatingDays = %d, erwartet 4", got)
	}
}

func TestComputeStatsNoMeasurements(t *testing.T) {
	j := &Jar{
		ID:              "g1",
		GrossFullWeight: 1200,
		TareWeight:      200,
		StartDate:       NewDate(2026, time.January, 1),
		TargetDate:      NewDate(2026, time.March, 1),
	}
	today := NewDate(2026, time.January, 1)
	s := ComputeStats(j, today)

	if !floatEquals(s.InitialNet, 1000) {
		t.Errorf("InitialNet = %v, erwartet 1000", s.InitialNet)
	}
	if !floatEquals(s.CurrentNet, 1000) {
		t.Errorf("CurrentNet = %v, erwartet 1000", s.CurrentNet)
	}
	if s.TargetStatus != TargetUnknown {
		t.Errorf("TargetStatus = %v, erwartet unbekannt", s.TargetStatus)
	}
	// 1000 g / 59 Tage
	wantPlanned := 1000.0 / 59.0
	if !floatEquals(s.PlannedDailyRate, wantPlanned) {
		t.Errorf("PlannedDailyRate = %v, erwartet %v", s.PlannedDailyRate, wantPlanned)
	}
}

func TestComputeStatsConsumption(t *testing.T) {
	j := &Jar{
		ID:              "g1",
		GrossFullWeight: 1200, // Inhalt 1000 g
		TareWeight:      200,
		StartDate:       NewDate(2026, time.January, 3), // Samstag
		TargetDate:      NewDate(2026, time.March, 3),
		Measurements: []Measurement{
			// erstes Wochenende: 30 g verbraucht -> brutto 1170
			{Date: NewDate(2026, time.January, 4), GrossWeight: 1170},
			// zweites Wochenende: weitere 30 g -> brutto 1140
			{Date: NewDate(2026, time.January, 11), GrossWeight: 1140},
		},
	}
	today := NewDate(2026, time.January, 11)
	s := ComputeStats(j, today)

	if !floatEquals(s.CurrentNet, 940) {
		t.Errorf("CurrentNet = %v, erwartet 940", s.CurrentNet)
	}
	if !floatEquals(s.TotalConsumed, 60) {
		t.Errorf("TotalConsumed = %v, erwartet 60", s.TotalConsumed)
	}
	if !floatEquals(s.ConsumedSinceLast, 30) {
		t.Errorf("ConsumedSinceLast = %v, erwartet 30", s.ConsumedSinceLast)
	}
	if len(s.Consumption) != 2 {
		t.Fatalf("len(Consumption) = %d, erwartet 2", len(s.Consumption))
	}
	// Start 03., letzte Messung 11. -> 8 Tage
	if s.DaysElapsed != 8 {
		t.Errorf("DaysElapsed = %d, erwartet 8", s.DaysElapsed)
	}
	if !floatEquals(s.BurnRatePerDay, 60.0/8.0) {
		t.Errorf("BurnRatePerDay = %v, erwartet %v", s.BurnRatePerDay, 60.0/8.0)
	}
	// Esstage Sa/So zwischen 03. und 11.: 03,04,10,11 = 4
	if !floatEquals(s.BurnRatePerEatingDay, 60.0/4.0) {
		t.Errorf("BurnRatePerEatingDay = %v, erwartet %v", s.BurnRatePerEatingDay, 60.0/4.0)
	}
	if s.EstimatedEmptyDate == nil {
		t.Fatal("EstimatedEmptyDate ist nil, erwartet einen Wert")
	}
	if !floatEquals(s.MaxConsumed, 30) || !floatEquals(s.MinConsumed, 30) {
		t.Errorf("Max/Min = %v/%v, erwartet 30/30", s.MaxConsumed, s.MinConsumed)
	}
}

func TestComputeStatsTargetMissed(t *testing.T) {
	// Sehr hoher Verbrauch -> Glas reicht nicht bis zum Zieldatum.
	j := &Jar{
		ID:              "g1",
		GrossFullWeight: 1200,
		TareWeight:      200,
		StartDate:       NewDate(2026, time.January, 1),
		TargetDate:      NewDate(2026, time.June, 1),
		Measurements: []Measurement{
			{Date: NewDate(2026, time.January, 11), GrossWeight: 700}, // 500 g in 10 Tagen
		},
	}
	today := NewDate(2026, time.January, 11)
	s := ComputeStats(j, today)

	if s.TargetStatus != TargetMissed {
		t.Errorf("TargetStatus = %v, erwartet nein", s.TargetStatus)
	}
	if s.TargetDiffDays >= 0 {
		t.Errorf("TargetDiffDays = %d, erwartet negativ", s.TargetDiffDays)
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		jar     Jar
		wantErr bool
	}{
		{"ok", Jar{ID: "g1", GrossFullWeight: 1200, TareWeight: 200,
			StartDate: NewDate(2026, 1, 1), TargetDate: NewDate(2026, 3, 1)}, false},
		{"leere id", Jar{ID: "", GrossFullWeight: 1200, TareWeight: 200}, true},
		{"tara zu groß", Jar{ID: "g1", GrossFullWeight: 200, TareWeight: 300}, true},
		{"ziel vor start", Jar{ID: "g1", GrossFullWeight: 1200, TareWeight: 200,
			StartDate: NewDate(2026, 3, 1), TargetDate: NewDate(2026, 1, 1)}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.jar.Validate()
			if (err != nil) != c.wantErr {
				t.Errorf("Validate() err = %v, wantErr = %v", err, c.wantErr)
			}
		})
	}
}

func TestDateJSONRoundtrip(t *testing.T) {
	d := NewDate(2026, time.February, 14)
	b, err := d.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"2026-02-14"` {
		t.Fatalf("MarshalJSON = %s, erwartet \"2026-02-14\"", b)
	}
	var d2 Date
	if err := d2.UnmarshalJSON(b); err != nil {
		t.Fatal(err)
	}
	if !d2.Equal(d.Time) {
		t.Fatalf("UnmarshalJSON = %v, erwartet %v", d2, d)
	}
}
