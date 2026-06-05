// Package web stellt die statischen Dateien der Web-UI als eingebettetes
// Dateisystem bereit, damit sie im Binary/Container mitgeliefert werden.
package web

import "embed"

// Files enthält die ausgelieferten statischen Dateien der Web-UI.
//
//go:embed index.html app.js style.css
var Files embed.FS
