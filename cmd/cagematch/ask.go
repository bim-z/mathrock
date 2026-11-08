package license

import (
	"errors"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"golang.org/x/term"
)

var errAbort = errors.New("cancelled")

func interactive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func run(field huh.Field) error {
	err := huh.NewForm(huh.NewGroup(field)).WithTheme(huh.ThemeCharm()).Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return errAbort
	}

	return err
}

func yes(title, hint string, start bool) (bool, error) {
	value := start

	err := run(huh.NewConfirm().Title(title).Description(hint).Affirmative("Yes").Negative("No").Value(&value))

	return value, err
}

type option struct {
	Key   string
	Label string
}

func choose(title, hint string, options []option, height int) (string, error) {
	var value string

	list := make([]huh.Option[string], 0, len(options))
	for _, entry := range options {
		list = append(list, huh.NewOption(entry.Label, entry.Key))
	}

	field := huh.NewSelect[string]().Title(title).Description(hint).Options(list...).Value(&value)
	if height > 0 {
		field = field.Height(height)
	}

	err := run(field)

	return value, err
}

func text(title, hint, start string, check func(string) error) (string, error) {
	var value string

	field := huh.NewInput().Title(title).Description(hint).Placeholder(start).Value(&value).Validate(func(entry string) error {
		entry = strings.TrimSpace(entry)

		if entry == "" {
			entry = start
		}

		if entry == "" {
			return errors.New("this field is required")
		}

		if check != nil {
			return check(entry)
		}

		return nil
	})

	err := run(field)

	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		value = start
	}

	return value, err
}

func advise() (string, error) {
	goal, err := choose("What matters most to you?", "", []option{
		{Key: "free", Label: "Anyone can do anything, no strings attached"},
		{Key: "simple", Label: "Simple and permissive, just keep my name on it"},
		{Key: "open", Label: "Changes must stay open source"},
	}, 0)
	if err != nil {
		return "", err
	}

	switch goal {
	case "free":
		kind, err := choose("What are you releasing?", "", []option{
			{Key: "code", Label: "Code"},
			{Key: "art", Label: "Docs, art, data or other creative work"},
		}, 0)
		if err != nil {
			return "", err
		}

		if kind == "art" {
			return "cc0-1.0", nil
		}

		return "unlicense", nil
	case "simple":
		patent, err := yes("Do you want an explicit patent grant?", "Useful when companies will contribute or depend on the code", false)
		if err != nil {
			return "", err
		}

		if patent {
			return "apache-2.0", nil
		}

		return "mit", nil
	}

	network, err := yes("Must people who run it as an online service share their changes too?", "Closes the loophole where hosted software never gets distributed", false)
	if err != nil {
		return "", err
	}

	if network {
		return "agpl-3.0", nil
	}

	linked, err := yes("May closed source programs link to your code?", "Typical for libraries that want wide adoption", false)
	if err != nil {
		return "", err
	}

	if !linked {
		return "gpl-3.0", nil
	}

	scope, err := choose("Which part must stay open?", "", []option{
		{Key: "mpl-2.0", Label: "Only the files I wrote, each file is covered separately"},
		{Key: "lgpl-3.0", Label: "The library itself, programs using it stay free to choose"},
	}, 0)

	return scope, err
}
