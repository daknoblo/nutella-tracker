package vision

import (
	"math"
	"testing"
)

func TestParseGrams(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{"reines json", `{"grams": 970, "confidence": 0.9}`, 970, false},
		{"json mit codefence", "```json\n{\"grams\": 985.5}\n```", 985.5, false},
		{"null grams", `{"grams": null}`, 0, true},
		{"negativ", `{"grams": -5}`, 0, true},
		{"freitext fallback", "Das Gewicht beträgt 942 g.", 942, false},
		{"kein wert", "kein Display erkennbar", 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseGrams(c.input)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, c.wantErr)
			}
			if !c.wantErr && math.Abs(got-c.want) > 1e-9 {
				t.Fatalf("got = %v, want %v", got, c.want)
			}
		})
	}
}

func TestConfigEnabled(t *testing.T) {
	if (Config{}).Enabled() {
		t.Error("leere Config sollte nicht enabled sein")
	}
	full := Config{Endpoint: "https://x", APIKey: "k", Model: "gpt-4o"}
	if !full.Enabled() {
		t.Error("vollständige Config sollte enabled sein")
	}
}

func TestFirstNumber(t *testing.T) {
	v, ok := firstNumber("ca. 12,5 kg")
	if !ok || math.Abs(v-12.5) > 1e-9 {
		t.Fatalf("firstNumber = %v, %v; erwartet 12.5, true", v, ok)
	}
}
