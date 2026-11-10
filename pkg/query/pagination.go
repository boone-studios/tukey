// Copyright (c) 2026 Boone Studios
// SPDX-License-Identifier: MIT

package query

import (
	"encoding/base64"
	"fmt"
	"strconv"
)

const (
	DefaultPageLimit = 50
	MaxPageLimit     = 100
	MaxContextDepth  = 5
)

type PageOptions struct {
	Limit     int
	Cursor    string
	Unbounded bool
}

type PageInfo struct {
	Limit      int    `json:"limit"`
	Returned   int    `json:"returned"`
	Total      int    `json:"total"`
	NextCursor string `json:"nextCursor,omitempty"`
}

func resolvePage(options PageOptions) (int, int, error) {
	offset := 0
	if options.Cursor != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(options.Cursor)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid cursor")
		}
		offset, err = strconv.Atoi(string(decoded))
		if err != nil || offset < 0 {
			return 0, 0, fmt.Errorf("invalid cursor")
		}
	}
	limit := options.Limit
	if limit < 0 && !options.Unbounded {
		return 0, 0, fmt.Errorf("limit must not be negative")
	}
	if options.Unbounded {
		return offset, -1, nil
	}
	if limit == 0 {
		limit = DefaultPageLimit
	}
	if limit > MaxPageLimit {
		limit = MaxPageLimit
	}
	return offset, limit, nil
}

func pageSlice[T any](items []T, offset, limit int) ([]T, bool) {
	if limit < 0 {
		return items, false
	}
	if offset >= len(items) {
		return []T{}, false
	}
	end := offset + limit
	if end >= len(items) {
		return items[offset:], false
	}
	return items[offset:end], true
}

func makePage(limit, offset, returned, total int, hasMore bool) *PageInfo {
	page := &PageInfo{Limit: limit, Returned: returned, Total: total}
	if hasMore {
		page.NextCursor = base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset + limit)))
	}
	return page
}
