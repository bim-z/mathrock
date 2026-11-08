/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package breadwinner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

type options struct {
	Out   string
	Paper string
	Look  string
	Yes   bool
}

func plural(count int, noun string) string {
	if count == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(count) + " " + noun + "s"
}

func sheetof(name string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "a4":
		return "a4", nil
	case "letter":
		return "letter", nil
	}

	return "", fmt.Errorf("unknown paper %q, use a4 or letter", name)
}

func looks() []string {
	return []string{"classic", "modern"}
}

func restyle(d *doc, name string) error {
	if name = strings.ToLower(strings.TrimSpace(name)); name != "" {
		d.Look = name
	}

	if d.Look == "" {
		d.Look = "classic"
	}

	for _, known := range looks() {
		if d.Look == known {
			return nil
		}
	}

	return fmt.Errorf("unknown template %q, use %s", d.Look, strings.Join(looks(), " or "))
}

func tagged(text, look string) string {
	if look == "" {
		return text
	}
	return strings.Replace(text, "---\n", "---\ntemplate: "+strings.ToLower(look)+"\n", 1)
}

func load(path string) (doc, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return doc{}, "", fmt.Errorf("%s does not exist, create one with 'resume new'", path)
		}
		return doc{}, "", err
	}

	raw := strings.ReplaceAll(strings.TrimPrefix(string(data), "\xef\xbb\xbf"), "\r\n", "\n")

	found, err := parse(raw)
	if err != nil {
		return doc{}, "", fmt.Errorf("%s: %w", path, err)
	}

	return found, raw, nil
}

func export(d doc, path, paper string) (int, []string, error) {
	var data []byte
	var pages int
	var notes []string
	var err error

	switch strings.ToLower(filepath.Ext(path)) {
	case ".pdf":
		data, pages, notes, err = rendered(d, paper)
	case ".docx":
		data, err = docx(d, paper)
	case ".txt":
		data = []byte(plaintext(d))
	default:
		return 0, nil, fmt.Errorf("cannot write %q, use a .pdf, .docx or .txt file", filepath.Base(path))
	}

	if err != nil {
		return 0, nil, err
	}

	return pages, notes, os.WriteFile(path, data, 0o644)
}

func vacant(path string, opt options) error {
	if _, err := os.Stat(path); err == nil && !opt.Yes {
		return fmt.Errorf("%s already exists, pass --yes to overwrite it", path)
	}
	return nil
}

func finish(d doc, raw, target string, opt options) error {
	paper, err := sheetof(opt.Paper)
	if err != nil {
		return err
	}

	if err := restyle(&d, opt.Look); err != nil {
		return err
	}

	pages, notes, err := export(d, target, paper)
	if err != nil {
		return err
	}

	line := "  " + pink.Render("saved") + " " + target

	if pages > 0 {
		line += faint.Render("  " + plural(pages, "page") + "  ·  " + strings.ToUpper(paper))
	}

	fmt.Println(line)

	for _, note := range notes {
		fmt.Println("  " + red.Render("note") + " " + note)
	}

	fmt.Println()
	fmt.Print(report(issues(d, raw, pages)))
	fmt.Println()

	return nil
}

func survey(opt options) error {
	paper, err := sheetof(opt.Paper)
	if err != nil {
		return err
	}

	opt.Paper = paper

	target := opt.Out
	if target == "" {
		target = "resume.pdf"
	}

	source := stem(target) + ".md"

	if err := vacant(source, opt); err != nil {
		return err
	}

	if err := vacant(target, opt); err != nil {
		return err
	}

	text, err := collect()
	if errors.Is(err, errAbort) {
		fmt.Println("\n  " + faint.Render("cancelled, nothing was saved") + "\n")
		return nil
	}
	if err != nil {
		return err
	}

	found, err := parse(text)
	if err != nil {
		return err
	}

	text = tagged(text, opt.Look)

	found, err = parse(text)
	if err != nil {
		return err
	}

	if err := os.WriteFile(source, []byte(text), 0o644); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("  " + pink.Render("saved") + " " + source + faint.Render("  edit it any time, then run 'resume build "+source+"'"))

	return finish(found, text, target, opt)
}

func scaffold(args []string, opt options) error {
	path := "resume.md"
	if len(args) > 0 {
		path = args[0]
	}

	if err := vacant(path, opt); err != nil {
		return err
	}

	if err := os.WriteFile(path, []byte(tagged(skeleton, opt.Look)), 0o644); err != nil {
		return err
	}

	fmt.Println("\n  " + pink.Render("created") + " " + path)
	fmt.Println("  " + faint.Render("replace the sample details, then run 'resume build "+path+"' or 'resume tui "+path+"'") + "\n")

	return nil
}

func build(path string, opt options) error {
	found, raw, err := load(path)
	if err != nil {
		return err
	}

	target := opt.Out
	if target == "" {
		target = stem(path) + ".pdf"
	}

	fmt.Println()

	return finish(found, raw, target, opt)
}

func display(path string, opt options) error {
	found, raw, err := load(path)
	if err != nil {
		return err
	}

	paper, err := sheetof(opt.Paper)
	if err != nil {
		return err
	}

	if err := restyle(&found, opt.Look); err != nil {
		return err
	}

	_, pages, _, err := rendered(found, paper)
	if err != nil {
		return err
	}

	lines, _ := preview(found, min(columns()-4, 88))

	fmt.Println()

	for _, line := range lines {
		fmt.Println("  " + line)
	}

	fmt.Println()
	fmt.Print(report(issues(found, raw, pages)))
	fmt.Println()

	return nil
}

func inspect(path string, opt options) error {
	found, raw, err := load(path)
	if err != nil {
		return err
	}

	paper, err := sheetof(opt.Paper)
	if err != nil {
		return err
	}

	if err := restyle(&found, opt.Look); err != nil {
		return err
	}

	_, pages, _, err := rendered(found, paper)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Print(report(issues(found, raw, pages)))
	fmt.Println()

	return nil
}

func Breadwinner() *cobra.Command {
	var opt options

	root := &cobra.Command{
		Use:          "resume",
		Short:        "Build an ATS friendly resume from a guided form or a markdown file",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return survey(opt)
		},
	}

	root.PersistentFlags().StringVarP(&opt.Paper, "paper", "p", "a4", "paper size, a4 or letter")
	root.PersistentFlags().StringVarP(&opt.Look, "template", "t", "", "resume template, classic or modern")
	root.Flags().StringVarP(&opt.Out, "out", "o", "", "output file ending in .pdf, .docx or .txt")
	root.Flags().BoolVarP(&opt.Yes, "yes", "y", false, "overwrite existing files")

	form := &cobra.Command{
		Use:   "form",
		Short: "Answer questions one by one and get a resume",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return survey(opt)
		},
	}

	form.Flags().StringVarP(&opt.Out, "out", "o", "", "output file ending in .pdf, .docx or .txt")
	form.Flags().BoolVarP(&opt.Yes, "yes", "y", false, "overwrite existing files")

	fresh := &cobra.Command{
		Use:   "new [file]",
		Short: "Write a ready to edit markdown template",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return scaffold(args, opt)
		},
	}

	fresh.Flags().BoolVarP(&opt.Yes, "yes", "y", false, "overwrite an existing file")

	making := &cobra.Command{
		Use:   "build <file.md>",
		Short: "Turn a markdown resume into a PDF, DOCX or TXT",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return build(args[0], opt)
		},
	}

	making.Flags().StringVarP(&opt.Out, "out", "o", "", "output file ending in .pdf, .docx or .txt")

	tui := &cobra.Command{
		Use:   "tui [file.md]",
		Short: "Preview, reorder, hide sections and export in a full screen view",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path := "resume.md"
			if len(args) > 0 {
				path = args[0]
			}

			paper, err := sheetof(opt.Paper)
			if err != nil {
				return err
			}

			return play(path, paper, opt.Look)
		},
	}

	showing := &cobra.Command{
		Use:   "show <file.md>",
		Short: "Print the resume in the terminal with its ATS check",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return display(args[0], opt)
		},
	}

	checking := &cobra.Command{
		Use:   "check <file.md>",
		Short: "Review a markdown resume for ATS problems",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return inspect(args[0], opt)
		},
	}

	root.AddCommand(form, fresh, making, tui, showing, checking)

	return root
}
