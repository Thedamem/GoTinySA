// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	ver "tinysa_go/internal/version"
)

//go:embed all:frontend/dist
var assets embed.FS

// logFilePath is set by setupLogging and shown in Settings → Debug.
var logFilePath string

// setupLogging writes diagnostics next to the user config:
// Linux: ~/.config/tinysa_go/tinysa_go.log, Windows: %AppData%\tinysa_go\tinysa_go.log.
func setupLogging() {
	dir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	dir = filepath.Join(dir, "tinysa_go")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	path := filepath.Join(dir, "tinysa_go.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	logFilePath = path
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("tinysa_go %s starting", ver.String())
}

func main() {
	setupLogging()
	app := NewApp()

	err := wails.Run(&options.App{
		Title:            "tinysa_go",
		Width:            1180,
		Height:           760,
		MinWidth:         900,
		MinHeight:        560,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 18, G: 20, B: 26, A: 1},
		OnStartup:        app.startup,
		OnBeforeClose:    app.beforeClose,
		Bind:             []interface{}{app},
	})
	if err != nil {
		log.Printf("fatal: %v", err)
		fmt.Fprintf(os.Stderr, "tinysa_go %s: %v\n", ver.String(), err)
		os.Exit(1)
	}
}
