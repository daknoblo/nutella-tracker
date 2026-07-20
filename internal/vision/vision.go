// Package vision kapselt die Anbindung an ein Microsoft-Foundry-/Azure-OpenAI-
// Vision-Modell, das aus einem Foto den Gewichtswert einer Waage ausliest.
//
// Es werden keine Bilder gespeichert; das Foto wird nur für die einzelne
// Anfrage als Base64-Data-URI an das Modell übermittelt.
package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Config bündelt die Verbindungsdaten zum Vision-Modell. Der API-Key wird
// ausschließlich aus einer Umgebungsvariablen befüllt und nie persistiert.
type Config struct {
	Endpoint   string // z. B. https://<resource>.openai.azure.com
	APIKey     string // Secret, nur aus Env-Variable
	Model      string // Deployment-/Modellname (z. B. gpt-4o)
	APIVersion string // z. B. 2024-08-01-preview
}

// Enabled meldet, ob alle nötigen Verbindungsdaten vorhanden sind.
func (c Config) Enabled() bool {
	return c.Endpoint != "" && c.APIKey != "" && c.Model != ""
}

// Client ruft das Vision-Modell auf.
type Client struct {
	cfg  Config
	http *http.Client
}

// New erzeugt einen Vision-Client mit sinnvollem HTTP-Timeout.
func New(cfg Config) *Client {
	if cfg.APIVersion == "" {
		cfg.APIVersion = "2024-08-01-preview"
	}
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: 60 * time.Second},
	}
}

// Enabled meldet, ob der Client einsatzbereit konfiguriert ist.
func (c *Client) Enabled() bool {
	return c.cfg.Enabled()
}

// Das Modell wird angewiesen, ausschließlich JSON in dieser Form zu liefern.
type modelResult struct {
	Grams      *float64 `json:"grams"`
	Confidence *float64 `json:"confidence"`
}

const systemPrompt = `Du bist ein Assistent, der das angezeigte Gewicht einer ` +
	`digitalen Waage von einem Foto abliest. Auf dem Foto steht ein Nutella-Glas ` +
	`auf einer Küchen-/Personenwaage. Lies ausschließlich den auf dem Display ` +
	`angezeigten Zahlenwert ab und gib ihn in Gramm an. ` +
	`Wenn die Anzeige in Kilogramm ist (z. B. 0,97 kg), rechne in Gramm um (970). ` +
	`Antworte ausschließlich mit JSON der Form {"grams": <Zahl|null>, "confidence": <0..1>}. ` +
	`Wenn du den Wert nicht sicher erkennst, setze grams auf null.`

// chatRequest/-Response bilden die für uns relevanten Felder der
// Azure-OpenAI-Chat-Completions-API ab.
type chatRequest struct {
	Messages       []chatMessage   `json:"messages"`
	MaxTokens      int             `json:"max_tokens"`
	Temperature    float64         `json:"temperature"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatMessage struct {
	Role    string        `json:"role"`
	Content []contentPart `json:"content"`
}

type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// RecognizeWeight schickt das Bild an das Vision-Modell und gibt das erkannte
// Gewicht in Gramm zurück. Ist kein Wert sicher erkennbar, wird ein Fehler
// zurückgegeben.
func (c *Client) RecognizeWeight(ctx context.Context, image []byte, contentType string) (float64, error) {
	if !c.Enabled() {
		return 0, errors.New("Vision-Erkennung ist nicht konfiguriert")
	}
	if len(image) == 0 {
		return 0, errors.New("kein Bild übergeben")
	}
	if contentType == "" {
		contentType = "image/jpeg"
	}

	dataURI := fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(image))

	reqBody := chatRequest{
		MaxTokens:      100,
		Temperature:    0,
		ResponseFormat: &responseFormat{Type: "json_object"},
		Messages: []chatMessage{
			{Role: "system", Content: []contentPart{{Type: "text", Text: systemPrompt}}},
			{Role: "user", Content: []contentPart{
				{Type: "text", Text: "Welches Gewicht zeigt die Waage an?"},
				{Type: "image_url", ImageURL: &imageURL{URL: dataURI}},
			}},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return 0, fmt.Errorf("Anfrage serialisieren: %w", err)
	}

	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		strings.TrimRight(c.cfg.Endpoint, "/"), c.cfg.Model, c.cfg.APIVersion)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("Anfrage erstellen: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", c.cfg.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("Vision-Dienst nicht erreichbar: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, fmt.Errorf("Antwort lesen: %w", err)
	}

	var cr chatResponse
	if err := json.Unmarshal(respBytes, &cr); err != nil {
		return 0, fmt.Errorf("Antwort parsen: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		if cr.Error != nil && cr.Error.Message != "" {
			return 0, fmt.Errorf("Vision-Dienst meldet Fehler: %s", cr.Error.Message)
		}
		return 0, fmt.Errorf("Vision-Dienst antwortete mit Status %d", resp.StatusCode)
	}
	if len(cr.Choices) == 0 {
		return 0, errors.New("Vision-Dienst lieferte keine Antwort")
	}

	grams, err := parseGrams(cr.Choices[0].Message.Content)
	if err != nil {
		return 0, err
	}
	return grams, nil
}

// parseGrams extrahiert den Gewichtswert aus der (JSON-)Antwort des Modells.
func parseGrams(content string) (float64, error) {
	content = strings.TrimSpace(content)
	// Etwaige Markdown-Codefences entfernen.
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var r modelResult
	if err := json.Unmarshal([]byte(content), &r); err != nil {
		// Fallback: erste Zahl aus dem Text ziehen.
		if v, ok := firstNumber(content); ok {
			return v, nil
		}
		return 0, fmt.Errorf("Antwort des Vision-Dienstes nicht verwertbar: %q", content)
	}
	if r.Grams == nil {
		return 0, errors.New("auf dem Foto wurde kein Gewicht erkannt")
	}
	if *r.Grams <= 0 {
		return 0, errors.New("erkanntes Gewicht ist nicht plausibel")
	}
	return *r.Grams, nil
}

// firstNumber liefert die erste im Text gefundene (Dezimal-)Zahl.
func firstNumber(s string) (float64, bool) {
	var b strings.Builder
	started := false
	for _, r := range s {
		if (r >= '0' && r <= '9') || (started && (r == '.' || r == ',')) {
			if r == ',' {
				b.WriteRune('.')
			} else {
				b.WriteRune(r)
			}
			started = true
			continue
		}
		if started {
			break
		}
	}
	if !started {
		return 0, false
	}
	v, err := strconv.ParseFloat(b.String(), 64)
	if err != nil {
		return 0, false
	}
	return v, true
}
