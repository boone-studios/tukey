// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package parser

import (
	"fmt"
	"sync"
)

// registry of available language parsers
var (
	mu       sync.RWMutex
	registry = map[string]LanguageParser{}
)

// Register adds a parser to the global registry.
// Typically called from parser init() functions.
func Register(p LanguageParser) {
	mu.Lock()
	defer mu.Unlock()

	lang := p.Language()
	if _, exists := registry[lang]; exists {
		panic(fmt.Sprintf("parser for language %q already registered", lang))
	}
	registry[lang] = p
}

// Get retrieves a parser for the given language key (e.g. "php").
func Get(language string) (LanguageParser, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := registry[language]
	return p, ok
}

// SupportedLanguages returns a list of registered language keys.
func SupportedLanguages() []string {
	mu.RLock()
	defer mu.RUnlock()

	langs := make([]string, 0, len(registry))
	for k := range registry {
		langs = append(langs, k)
	}
	return langs
}
