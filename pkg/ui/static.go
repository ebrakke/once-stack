// Package ui provides shared UI assets and rendering helpers for ONCE apps.
package ui

import "embed"

// StaticFS contains compiled shared UI assets.
//
//go:embed static/*
var StaticFS embed.FS

// OnceCSS is the compiled Tailwind stylesheet shared by ONCE apps.
//
//go:embed static/once.css
var OnceCSS []byte
