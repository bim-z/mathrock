/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package gaza

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const limit = 2 << 20

var (
	title = lipgloss.NewStyle().Bold(true).Underline(true)
	faint = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	red   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

type options struct {
	Plain   bool
	Number  bool
	Squeeze bool
	Syntax  bool
	Theme   string
	Style   *chroma.Style
	Format  chroma.Formatter
}

func human(size int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(size)
	idx := 0

	for value >= 1024 && idx < len(units)-1 {
		value /= 1024
		idx++
	}

	if idx == 0 {
		return fmt.Sprintf("%d B", size)
	}

	return fmt.Sprintf("%.1f %s", value, units[idx])
}

func noun(count int) string {
	if count == 1 {
		return "1 line"
	}
	return strconv.Itoa(count) + " lines"
}

func tally(path string) int {
	handle, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer handle.Close()

	buf := make([]byte, 32<<10)
	count := 0

	var last byte

	for {
		size, err := handle.Read(buf)
		if size > 0 {
			count += bytes.Count(buf[:size], []byte{'\n'})
			last = buf[size-1]
		}
		if err != nil {
			break
		}
	}

	if last != 0 && last != '\n' {
		count++
	}

	return count
}

func expand(text string, col *int, tint bool) string {
	var out strings.Builder

	for _, char := range text {
		switch {
		case char == '\t':
			gap := 4 - *col%4
			out.WriteString(strings.Repeat(" ", gap))
			*col += gap
		case char < 0x20 || char == 0x7f:
			mark := "^" + string(char^0x40)
			if tint {
				mark = faint.Render(mark)
			}
			out.WriteString(mark)
			*col += 2
		default:
			out.WriteRune(char)
			*col++
		}
	}

	return out.String()
}

func clean(line string) string {
	col := 0
	return expand(line, &col, true)
}

func depth() string {
	kind := os.Getenv("TERM")
	color := strings.ToLower(os.Getenv("COLORTERM"))

	switch {
	case os.Getenv("NO_COLOR") != "" || kind == "dumb":
		return ""
	case strings.Contains(color, "truecolor") || strings.Contains(color, "24bit") || os.Getenv("WT_SESSION") != "":
		return "terminal16m"
	case kind == "" || strings.Contains(kind, "256color"):
		return "terminal256"
	}

	return "terminal16"
}

func pick(name, text string) chroma.Lexer {
	lexer := lexers.Match(filepath.Base(name))
	if lexer == nil {
		lexer = lexers.Analyse(text)
	}
	if lexer == nil || lexer.Config().Name == "plaintext" {
		return nil
	}

	return chroma.Coalesce(lexer)
}

func paint(lexer chroma.Lexer, opt options, text string) []string {
	iter, err := lexer.Tokenise(nil, text)
	if err != nil {
		return nil
	}

	rows := chroma.SplitTokensIntoLines(iter.Tokens())
	raw := strings.Split(text, "\n")
	out := make([]string, len(rows))

	for idx, row := range rows {
		if idx >= len(raw) {
			break
		}

		var joined strings.Builder
		for _, tok := range row {
			joined.WriteString(tok.Value)
		}

		if strings.TrimSuffix(joined.String(), "\n") != raw[idx] {
			continue
		}

		col := 0
		fixed := make([]chroma.Token, 0, len(row))

		for pos, tok := range row {
			value := tok.Value
			if pos == len(row)-1 {
				value = strings.TrimSuffix(value, "\n")
			}
			if value == "" {
				continue
			}
			fixed = append(fixed, chroma.Token{Type: tok.Type, Value: expand(value, &col, false)})
		}

		var buf strings.Builder
		if err := opt.Format.Format(&buf, opt.Style, chroma.Literator(fixed...)); err != nil {
			continue
		}

		out[idx] = buf.String()
	}

	return out
}

func slurp(in *bufio.Reader, size int64) (string, *bufio.Reader, bool) {
	if size > limit {
		return "", in, false
	}

	if size >= 0 {
		data, err := io.ReadAll(in)
		if err != nil {
			return "", bufio.NewReader(bytes.NewReader(data)), false
		}
		return string(data), bufio.NewReader(bytes.NewReader(data)), true
	}

	data, err := io.ReadAll(io.LimitReader(in, limit+1))
	if err != nil || len(data) > limit {
		return "", bufio.NewReader(io.MultiReader(bytes.NewReader(data), in)), false
	}

	return string(data), bufio.NewReader(bytes.NewReader(data)), true
}

func copying(out io.Writer, in *bufio.Reader, squeeze bool) error {
	if !squeeze {
		_, err := io.Copy(out, in)
		return err
	}

	var prev bool

	for {
		line, err := in.ReadString('\n')
		if line != "" {
			empty := line == "\n"
			if !(empty && prev) {
				if _, werr := io.WriteString(out, line); werr != nil {
					return werr
				}
			}
			prev = empty
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func render(out *bufio.Writer, in *bufio.Reader, wide int, opt options, colored []string) error {
	count := 0
	row := 0

	var prev bool

	for {
		line, err := in.ReadString('\n')
		if line != "" {
			line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
			skip := opt.Squeeze && line == "" && prev
			prev = line == ""

			if !skip {
				count++
				text := clean(line)
				if row < len(colored) && colored[row] != "" {
					text = colored[row]
				}

				lead := "  "
				if opt.Number {
					lead = "  " + faint.Render(fmt.Sprintf("%*d", wide, count)) + " " + faint.Render("│")
					if text != "" {
						lead += " "
					}
				}

				if _, werr := io.WriteString(out, lead+text+"\n"); werr != nil {
					return werr
				}
			}

			row++
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func block(out *bufio.Writer, name string, in *bufio.Reader, size int64, lines int, opt options) error {
	head, _ := in.Peek(8000)

	if bytes.IndexByte(head, 0) >= 0 {
		parts := []string{"binary"}
		if size >= 0 {
			parts = []string{human(size), "binary"}
		}

		fmt.Fprintln(out, "  "+title.Render(name)+" "+faint.Render(strings.Join(parts, " · ")))
		fmt.Fprintln(out, "\n  "+faint.Render("binary file, content not shown"))

		return nil
	}

	var colored []string
	var lang string

	if opt.Syntax && opt.Style != nil && opt.Format != nil {
		text, rest, whole := slurp(in, size)
		in = rest

		if whole {
			if lines == 0 && text != "" {
				lines = strings.Count(text, "\n")
				if !strings.HasSuffix(text, "\n") {
					lines++
				}
			}

			if lexer := pick(name, text); lexer != nil {
				lang = lexer.Config().Name
				colored = paint(lexer, opt, strings.ReplaceAll(text, "\r\n", "\n"))
			}
		}
	}

	var parts []string

	if lang != "" {
		parts = append(parts, lang)
	}

	switch {
	case size == 0:
		parts = append(parts, "empty")
	case size > 0:
		parts = append(parts, human(size))
	}

	if lines > 0 {
		parts = append(parts, noun(lines))
	}

	top := "  " + title.Render(name)
	if len(parts) > 0 {
		top += " " + faint.Render(strings.Join(parts, " · "))
	}

	fmt.Fprintln(out, top)
	fmt.Fprintln(out)

	return render(out, in, max(3, len(strconv.Itoa(lines))), opt, colored)
}

func open(out *bufio.Writer, path string, opt options, plain bool, first bool) error {
	handle, err := os.Open(path)
	if err != nil {
		return err
	}
	defer handle.Close()

	info, err := handle.Stat()
	if err != nil {
		return err
	}

	if info.IsDir() {
		return fmt.Errorf("%s: is a directory", path)
	}

	in := bufio.NewReaderSize(handle, 64<<10)

	if plain {
		return copying(out, in, opt.Squeeze)
	}

	if !first {
		fmt.Fprintln(out)
	}

	return block(out, path, in, info.Size(), tally(path), opt)
}

func emit(w, fault io.Writer, in io.Reader, args []string, opt options, tty bool) error {
	out := bufio.NewWriter(w)
	plain := opt.Plain || !tty

	if len(args) == 0 {
		args = []string{"-"}
	}

	if tty && !plain {
		fmt.Fprintln(out)
	}

	var bad int

	for idx, path := range args {
		var err error

		if path == "-" {
			reader := bufio.NewReaderSize(in, 64<<10)
			if plain {
				err = copying(out, reader, opt.Squeeze)
			} else {
				if idx > 0 {
					fmt.Fprintln(out)
				}
				err = block(out, "stdin", reader, -1, 0, opt)
			}
		} else {
			err = open(out, path, opt, plain, idx == 0)
		}

		if err != nil {
			if ferr := out.Flush(); ferr != nil {
				return ferr
			}
			bad++
			fmt.Fprintln(fault, red.Render("dog: "+err.Error()))
		}
	}

	if tty && !plain {
		fmt.Fprintln(out)
	}

	if err := out.Flush(); err != nil {
		return err
	}

	if bad > 0 {
		return fmt.Errorf("%d of %d inputs could not be read", bad, len(args))
	}

	return nil
}

func setup(opt *options) error {
	name := opt.Theme
	if name == "" {
		name = "monokai"
		if !lipgloss.HasDarkBackground() {
			name = "github"
		}
	}

	style, ok := styles.Registry[name]
	if !ok {
		names := styles.Names()
		sort.Strings(names)
		return fmt.Errorf("unknown theme %q, choose one of: %s", name, strings.Join(names, ", "))
	}

	kind := depth()
	if kind == "" {
		return nil
	}

	opt.Style = style
	opt.Format = formatters.Get(kind)

	return nil
}

func Catbamboo() *cobra.Command {
	var opt options

	cmd := &cobra.Command{
		Use:          "catbamboo [file...]",
		Short:        "Print files with a clean and readable layout",
		SilenceUsage: true,
		Args:         cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && term.IsTerminal(int(os.Stdin.Fd())) {
				return cmd.Help()
			}

			tty := term.IsTerminal(int(os.Stdout.Fd()))

			if opt.Syntax && tty && !opt.Plain {
				if err := setup(&opt); err != nil {
					return err
				}
			}

			err := emit(os.Stdout, os.Stderr, os.Stdin, args, opt, tty)
			if errors.Is(err, syscall.EPIPE) {
				return nil
			}

			return err
		},
	}

	cmd.Flags().BoolVarP(&opt.Plain, "plain", "p", false, "Print raw content like cat")
	cmd.Flags().BoolVarP(&opt.Number, "number", "n", true, "Show line numbers, use --number=false to hide them")
	cmd.Flags().BoolVar(&opt.Syntax, "syntax", true, "Color code by file type, use --syntax=false to turn it off")
	cmd.Flags().StringVarP(&opt.Theme, "theme", "t", "", "Color theme such as monokai, dracula or github")
	cmd.Flags().BoolVarP(&opt.Squeeze, "squeeze", "s", false, "Collapse repeated empty lines")

	return cmd
}
