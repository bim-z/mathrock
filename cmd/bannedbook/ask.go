/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package bannedbook

import (
	"errors"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"golang.org/x/term"
)

type prompt struct {
	plain bool
}

func newprompt() prompt {
	return prompt{plain: !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd()))}
}

func (p prompt) run(field huh.Field) error {
	if p.plain {
		return errtty
	}

	form := huh.NewForm(huh.NewGroup(field)).WithTheme(huh.ThemeCharm())

	err := form.Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return errAbort
	}

	return err
}

func (p prompt) yes(title, hint string) (bool, error) {
	value := true

	err := p.run(huh.NewConfirm().Title(title).Description(hint).Affirmative("Yes").Negative("No").Value(&value))

	return value, err
}

func (p prompt) choose(title string, options []option) (string, error) {
	var value string

	list := make([]huh.Option[string], 0, len(options))
	for _, entry := range options {
		list = append(list, huh.NewOption(entry.Label, entry.Key))
	}

	err := p.run(huh.NewSelect[string]().Title(title).Options(list...).Value(&value))

	return value, err
}

func (p prompt) text(title, hint, start string, need bool, check func(string) error) (string, error) {
	var value string

	field := huh.NewInput().Title(title).Description(hint).Placeholder(start).Value(&value).Validate(func(entry string) error {
		entry = strings.TrimSpace(entry)

		if entry == "" && need && start == "" {
			return errors.New("this field is required")
		}

		if entry != "" && check != nil {
			return check(entry)
		}

		return nil
	})

	err := p.run(field)

	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		value = start
	}

	return value, err
}
