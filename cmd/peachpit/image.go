/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package peachpit

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	off   = "off"
	auto  = "auto"
	kitty = "kitty"
	sixel = "sixel"
	iterm = "iterm"
	wipe  = "\x1b_Ga=d,d=A,q=2\x1b\\"
)

type picture struct {
	Data []byte
	Wide int
	High int
}

type found struct {
	pic *picture
	err error
}

var (
	gate   sync.Mutex
	shelf  = map[string]found{}
	cooked = map[string]string{}
	client = &http.Client{Timeout: 10 * time.Second}
)

func forget() {
	gate.Lock()
	defer gate.Unlock()
	shelf = map[string]found{}
	cooked = map[string]string{}
}

func remote(src string) ([]byte, error) {
	resp, err := client.Get(src)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	return io.ReadAll(io.LimitReader(resp.Body, 32<<20))
}

func local(base, src string) ([]byte, error) {
	path := src
	if !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}

	data, err := os.ReadFile(path)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		return data, err
	}

	plain, perr := url.PathUnescape(src)
	if perr != nil || plain == src {
		return nil, err
	}

	if !filepath.IsAbs(plain) {
		plain = filepath.Join(base, plain)
	}

	return os.ReadFile(plain)
}

func read(base, src string) (*picture, error) {
	var data []byte
	var err error

	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		data, err = remote(src)
	} else {
		data, err = local(base, src)
	}
	if err != nil {
		return nil, err
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New("unsupported image format")
	}

	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width*cfg.Height > 100_000_000 {
		return nil, errors.New("image size is not supported")
	}

	return &picture{Data: data, Wide: cfg.Width, High: cfg.Height}, nil
}

func source(base, src string) (*picture, error) {
	key := base + "\x00" + src

	gate.Lock()
	hit, ok := shelf[key]
	gate.Unlock()

	if ok {
		return hit.pic, hit.err
	}

	pic, err := read(base, src)

	gate.Lock()
	shelf[key] = found{pic: pic, err: err}
	gate.Unlock()

	return pic, err
}

func fit(pic *picture, cols, rows, cx, cy int) (int, int, int, int) {
	scale := 1.0

	if w := float64(cols*cx) / float64(pic.Wide); w < scale {
		scale = w
	}
	if h := float64(rows*cy) / float64(pic.High); h < scale {
		scale = h
	}

	wide := max(1, int(float64(pic.Wide)*scale))
	high := max(1, int(float64(pic.High)*scale))

	return (wide + cx - 1) / cx, (high + cy - 1) / cy, wide, high
}

func resize(src image.Image, wide, high int) *image.NRGBA {
	box := src.Bounds()
	sw, sh := box.Dx(), box.Dy()
	dst := image.NewNRGBA(image.Rect(0, 0, wide, high))

	for y := 0; y < high; y++ {
		y0 := y * sh / high
		y1 := max(y0+1, (y+1)*sh/high)

		for x := 0; x < wide; x++ {
			x0 := x * sw / wide
			x1 := max(x0+1, (x+1)*sw/wide)

			var sr, sg, sb, sa, count uint64

			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					r, g, b, a := src.At(box.Min.X+sx, box.Min.Y+sy).RGBA()
					sr += uint64(r)
					sg += uint64(g)
					sb += uint64(b)
					sa += uint64(a)
					count++
				}
			}

			if sa == 0 {
				continue
			}

			dst.SetNRGBA(x, y, color.NRGBA{
				R: uint8(sr * 255 / sa),
				G: uint8(sg * 255 / sa),
				B: uint8(sb * 255 / sa),
				A: uint8((sa / count) >> 8),
			})
		}
	}

	return dst
}

func bytesof(img image.Image) ([]byte, error) {
	var buf bytes.Buffer

	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func kgp(img image.Image) (string, error) {
	raw, err := bytesof(img)
	if err != nil {
		return "", err
	}

	data := base64.StdEncoding.EncodeToString(raw)

	var out strings.Builder

	for first := true; len(data) > 0; first = false {
		size := min(4096, len(data))
		more := 1
		if size == len(data) {
			more = 0
		}

		if first {
			fmt.Fprintf(&out, "\x1b_Ga=T,f=100,q=2,C=1,m=%d;%s\x1b\\", more, data[:size])
		} else {
			fmt.Fprintf(&out, "\x1b_Gm=%d;%s\x1b\\", more, data[:size])
		}

		data = data[size:]
	}

	return out.String(), nil
}

func inline(img *image.NRGBA) (string, error) {
	raw, err := bytesof(img)
	if err != nil {
		return "", err
	}

	box := img.Bounds()

	return fmt.Sprintf(
		"\x1b]1337;File=inline=1;doNotMoveCursor=1;size=%d;width=%dpx;height=%dpx;preserveAspectRatio=1:%s\a",
		len(raw), box.Dx(), box.Dy(), base64.StdEncoding.EncodeToString(raw),
	), nil
}

func run(out *strings.Builder, char byte, count int) {
	if count > 3 {
		fmt.Fprintf(out, "!%d%c", count, char)
		return
	}

	for i := 0; i < count; i++ {
		out.WriteByte(char)
	}
}

func sextet(img *image.NRGBA) string {
	box := img.Bounds()
	wide, high := box.Dx(), box.Dy()

	pal := image.NewPaletted(box, palette.Plan9)
	draw.FloydSteinberg.Draw(pal, box, img, box.Min)

	var out strings.Builder

	out.WriteString("\x1bP9;1q")
	fmt.Fprintf(&out, "\"1;1;%d;%d", wide, high)

	for i, c := range palette.Plan9 {
		r, g, b, _ := c.RGBA()
		fmt.Fprintf(&out, "#%d;2;%d;%d;%d", i, r*100/65535, g*100/65535, b*100/65535)
	}

	for top := 0; top < high; top += 6 {
		var bands [256][]byte

		for dy := 0; dy < 6 && top+dy < high; dy++ {
			for x := 0; x < wide; x++ {
				if img.NRGBAAt(box.Min.X+x, box.Min.Y+top+dy).A < 128 {
					continue
				}

				idx := pal.ColorIndexAt(box.Min.X+x, box.Min.Y+top+dy)
				if bands[idx] == nil {
					bands[idx] = make([]byte, wide)
				}
				bands[idx][x] |= 1 << uint(dy)
			}
		}

		for idx, bits := range bands {
			if bits == nil {
				continue
			}

			fmt.Fprintf(&out, "#%d", idx)

			last := bits[0] + 63
			count := 0

			for _, bit := range bits {
				char := bit + 63
				if char == last {
					count++
					continue
				}
				run(&out, last, count)
				last, count = char, 1
			}

			run(&out, last, count)
			out.WriteByte('$')
		}

		out.WriteByte('-')
	}

	out.WriteString("\x1b\\")

	return out.String()
}

func encode(proto string, pic *picture, wide, high int) (string, error) {
	key := fmt.Sprintf("%s|%p|%d|%d", proto, pic, wide, high)

	gate.Lock()
	hit, ok := cooked[key]
	gate.Unlock()

	if ok {
		return hit, nil
	}

	img, _, err := image.Decode(bytes.NewReader(pic.Data))
	if err != nil {
		return "", err
	}

	small := resize(img, wide, high)

	var out string

	switch proto {
	case kitty:
		out, err = kgp(small)
	case iterm:
		out, err = inline(small)
	case sixel:
		out = sextet(small)
	default:
		err = errors.New("images are disabled")
	}
	if err != nil {
		return "", err
	}

	gate.Lock()
	cooked[key] = out
	gate.Unlock()

	return out, nil
}
