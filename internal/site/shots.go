package site

// Shot beschreibt einen automatisiert aufgenommenen Screenshot der Demo-Seite.
// Die Liste ist die einzige Quelle der Wahrheit: Der Screenshot-Runner liest
// sie aus screenshots.json, die Galerie der Projektseite rendert sie mit.
type Shot struct {
	// Name ist der Dateiname ohne Endung.
	Name string `json:"name"`
	// Title ist die Bildunterschrift in der Galerie.
	Title string `json:"title"`
	// Selector schränkt die Aufnahme auf ein Element ein (leer = ganze Seite).
	Selector string `json:"selector,omitempty"`
	// Width/Height ist das Browser-Fenster in CSS-Pixeln.
	Width  int `json:"width"`
	Height int `json:"height"`
	// FullPage nimmt die komplette scrollbare Seite auf.
	FullPage bool `json:"fullPage,omitempty"`
	// SelectJar wählt vor der Aufnahme ein anderes Glas aus.
	SelectJar string `json:"selectJar,omitempty"`
	// Click klickt vor der Aufnahme auf das angegebene Element.
	Click string `json:"click,omitempty"`
}

const (
	desktopWidth  = 1280
	desktopHeight = 900
	// Unterhalb von 800 px stehen die Karten einspaltig – so wird die
	// Einstellungs-Karte nicht auf die Höhe der Nachbarspalte gestreckt.
	sectionWidth = 760
	mobileWidth  = 420
	mobileHeight = 900
)

// Shots liefert alle Screenshots, die für Doku und Galerie erzeugt werden.
// Sie decken jede Sektion der Web-UI mit Demo-Inhalten ab.
func Shots() []Shot {
	return []Shot{
		{
			Name:     "overview",
			Title:    "Gesamtübersicht",
			Width:    desktopWidth,
			Height:   desktopHeight,
			FullPage: true,
		},
		{
			Name:     "timeline",
			Title:    "Zeitlicher Verlauf: Fortschritt zwischen Start und Zieldatum",
			Selector: "#timelineSection",
			Width:    desktopWidth,
			Height:   desktopHeight,
		},
		{
			Name:     "stats",
			Title:    "Statistik: Inhalt, Extremwerte, Burnrate und Prognose",
			Selector: "#statsSection",
			Width:    desktopWidth,
			Height:   desktopHeight,
		},
		{
			Name:      "status-missed",
			Title:     "Soll/Ist-Abgleich: Warnung, wenn das Glas nicht bis zum Zieldatum reicht",
			Selector:  "#statsSection",
			Width:     desktopWidth,
			Height:    desktopHeight,
			SelectJar: "demo-urlaub",
		},
		{
			Name:     "chart-content",
			Title:    "Restinhalt über die Zeit: Ist-Kurve gegen Soll-Linie",
			Selector: "#chartsSection .card:nth-of-type(1)",
			Width:    desktopWidth,
			Height:   desktopHeight,
		},
		{
			Name:     "chart-projection",
			Title:    "Projektion des aktuellen Verbrauchs bis zum Zieldatum",
			Selector: "#chartsSection .card:nth-of-type(2)",
			Width:    desktopWidth,
			Height:   desktopHeight,
		},
		{
			Name:     "chart-consumption",
			Title:    "Verbrauch pro Messung",
			Selector: "#chartsSection .card:nth-of-type(3)",
			Width:    desktopWidth,
			Height:   desktopHeight,
		},
		{
			Name:     "measurements",
			Title:    "Weigh-In inkl. optionaler Foto-Erkennung und Messhistorie",
			Selector: "#measureSection",
			Width:    desktopWidth,
			Height:   desktopHeight,
		},
		{
			Name:     "settings",
			Title:    "Einstellungen: Glasgröße, Tara, Start- und Zieldatum",
			Selector: "#settingsSection",
			Width:    sectionWidth,
			Height:   desktopHeight,
		},
		{
			Name:     "new-jar",
			Title:    "Neues Glas anlegen",
			Selector: "#settingsSection",
			Width:    sectionWidth,
			Height:   desktopHeight,
			Click:    "#newJarBtn",
		},
		{
			Name:     "mobile",
			Title:    "Mobile Ansicht",
			Width:    mobileWidth,
			Height:   mobileHeight,
			FullPage: true,
		},
	}
}
