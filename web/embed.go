package web

import "embed"

// Content contains the self-hosted browser workspace.
//
//go:embed index.html styles.css app.js
var Content embed.FS
