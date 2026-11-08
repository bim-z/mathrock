/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package algernon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const agent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"

var client = &http.Client{Timeout: 20 * time.Second}

var (
	missing = errors.New("not found")
	blocked = errors.New("blocked or rate limited")
	noimage = errors.New("not an image")
)

var types = map[string]string{
	"image/png":     ".png",
	"image/jpeg":    ".jpg",
	"image/jpg":     ".jpg",
	"image/webp":    ".webp",
	"image/gif":     ".gif",
	"image/svg+xml": ".svg",
	"image/avif":    ".avif",
}

var (
	first  = regexp.MustCompile(`(?i)<meta[^>]+(?:property|name)=["'](?:og|twitter):image(?::secure_url)?["'][^>]*?content=["']([^"']+)["']`)
	second = regexp.MustCompile(`(?i)<meta[^>]+content=["']([^"']+)["'][^>]*?(?:property|name)=["'](?:og|twitter):image(?::secure_url)?["']`)
)

type result struct {
	site site
	path string
	size int
	err  error
}

func expand(link, user string) string {
	handle := user
	if !strings.Contains(handle, ".") {
		handle += ".bsky.social"
	}

	replacer := strings.NewReplacer(
		"{u}", url.PathEscape(user),
		"{q}", url.QueryEscape(user),
		"{b}", url.QueryEscape(handle),
	)

	return replacer.Replace(link)
}

func fetch(ctx context.Context, method, link string, body []byte) ([]byte, string, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, link, reader)
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("User-Agent", agent)
	req.Header.Set("Accept", "*/*")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if key := os.Getenv("UNAVATAR_KEY"); key != "" && strings.HasPrefix(link, "https://unavatar.io/") {
		req.Header.Set("x-api-key", key)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound, http.StatusGone:
		return nil, "", missing
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests:
		return nil, "", blocked
	default:
		return nil, "", fmt.Errorf("status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 12<<20))
	if err != nil {
		return nil, "", err
	}

	return data, resp.Header.Get("Content-Type"), nil
}

func dig(node any, path string) string {
	for _, key := range strings.Split(path, ".") {
		switch value := node.(type) {
		case map[string]any:
			node = value[key]
		case []any:
			index, err := strconv.Atoi(key)
			if err != nil || index < 0 || index >= len(value) {
				return ""
			}

			node = value[index]
		default:
			return ""
		}
	}

	text, _ := node.(string)

	return text
}

func lookup(body []byte, path string) (string, error) {
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return "", errors.New("unexpected response")
	}

	for _, option := range strings.Split(path, "|") {
		if found := dig(root, option); found != "" {
			return found, nil
		}
	}

	return "", missing
}

func scrape(markup string) string {
	if match := first.FindStringSubmatch(markup); match != nil {
		return match[1]
	}

	if match := second.FindStringSubmatch(markup); match != nil {
		return match[1]
	}

	return ""
}

func locate(ctx context.Context, item site, user string) (string, error) {
	link := expand(item.link, user)

	var ref string

	switch item.kind {
	case direct:
		return link, nil
	case page:
		body, _, err := fetch(ctx, http.MethodGet, link, nil)
		if err != nil {
			return "", err
		}

		ref = scrape(string(body))
	case api:
		body, _, err := fetch(ctx, http.MethodGet, link, nil)
		if err != nil {
			return "", err
		}

		if ref, err = lookup(body, item.path); err != nil {
			return "", err
		}
	case graph:
		payload, err := json.Marshal(map[string]any{
			"query":     item.query,
			"variables": map[string]any{item.field: user},
		})
		if err != nil {
			return "", err
		}

		body, _, err := fetch(ctx, http.MethodPost, link, payload)
		if err != nil {
			return "", err
		}

		if ref, err = lookup(body, item.path); err != nil {
			return "", err
		}
	case regex:
		body, _, err := fetch(ctx, http.MethodGet, link, nil)
		if err != nil {
			return "", err
		}

		if match := item.find.FindSubmatch(body); match != nil {
			ref = string(match[1])
		}
	}

	ref = html.UnescapeString(strings.TrimSpace(ref))
	if ref == "" || (item.skip != "" && strings.Contains(ref, item.skip)) {
		return "", missing
	}

	base, err := url.Parse(link)
	if err != nil {
		return "", err
	}

	target, err := base.Parse(ref)
	if err != nil {
		return "", err
	}

	return target.String() + item.tail, nil
}

func extension(ctype string) string {
	name, _, _ := mime.ParseMediaType(ctype)
	if value, ok := types[name]; ok {
		return value
	}

	return ".img"
}

func check(ctx context.Context, item site, user, dir string) result {
	ctx, stop := context.WithTimeout(ctx, 25*time.Second)
	defer stop()

	link, err := locate(ctx, item, user)
	if err != nil {
		return result{site: item, err: err}
	}

	data, ctype, err := fetch(ctx, http.MethodGet, link, nil)
	if err != nil {
		return result{site: item, err: err}
	}

	if !strings.HasPrefix(ctype, "image/") {
		return result{site: item, err: noimage}
	}

	if len(data) < 100 {
		return result{site: item, err: missing}
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return result{site: item, err: err}
	}

	file := filepath.Join(dir, item.slug+extension(ctype))
	if err := os.WriteFile(file, data, 0644); err != nil {
		return result{site: item, err: err}
	}

	return result{site: item, path: file, size: len(data)}
}

func sweep(ctx context.Context, list []site, user, dir string, workers int, tick func(int)) []result {
	jobs := make(chan int)
	results := make([]result, len(list))

	var group sync.WaitGroup
	var lock sync.Mutex

	done := 0

	for range max(workers, 1) {
		group.Add(1)

		go func() {
			defer group.Done()

			for index := range jobs {
				results[index] = check(ctx, list[index], user, dir)

				lock.Lock()
				done++
				tick(done)
				lock.Unlock()
			}
		}()
	}

	for index := range list {
		jobs <- index
	}

	close(jobs)
	group.Wait()

	return results
}
