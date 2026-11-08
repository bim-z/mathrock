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
	"strings"
	"time"
)

var (
	errAbort = errors.New("cancelled")
	errtty   = errors.New("clarifying needs an interactive terminal, use gtd add -i to only capture")
)

type option struct {
	Key   string
	Label string
}

type asker interface {
	yes(title, hint string) (bool, error)
	choose(title string, options []option) (string, error)
	text(title, hint, start string, need bool, check func(string) error) (string, error)
}

type verdict struct {
	List    string
	Text    string
	Who     string
	Context string
	Due     time.Time
	Clock   bool
	Trash   bool
	Follow  *verdict
}

func valid(value string) error {
	_, _, err := moment(value, time.Now())
	return err
}

func clarify(a asker, text string) (verdict, error) {
	actionable, err := a.yes("Is it actionable?", "Can you actually do something about it?")
	if err != nil {
		return verdict{}, err
	}

	if !actionable {
		kind, err := a.choose("What is it then?", []option{
			{Key: someday, Label: "Someday/Maybe, I might want to do it later"},
			{Key: reference, Label: "Reference, information worth keeping"},
			{Key: "trash", Label: "Trash, it is not needed"},
		})
		if err != nil {
			return verdict{}, err
		}

		if kind == "trash" {
			return verdict{Trash: true, Text: text}, nil
		}

		return verdict{List: kind, Text: text}, nil
	}

	many, err := a.yes("Does it take more than one step?", "A multi step outcome is a project")
	if err != nil {
		return verdict{}, err
	}

	if !many {
		return route(a, text, true)
	}

	outcome, err := a.text("Desired outcome", "What does done look like?", text, true, nil)
	if err != nil {
		return verdict{}, err
	}

	step, err := a.text("Next action", "The very next physical step, starting with a verb", "", true, nil)
	if err != nil {
		return verdict{}, err
	}

	follow, err := route(a, step, false)
	if err != nil {
		return verdict{}, err
	}

	return verdict{List: projects, Text: outcome, Follow: &follow}, nil
}

func route(a asker, text string, quick bool) (verdict, error) {
	if quick {
		fast, err := a.yes("Does it take less than two minutes?", "If so, do it right now")
		if err != nil {
			return verdict{}, err
		}

		if fast {
			over, err := a.yes("Go ahead and do it. Is it finished?", "Answer no to keep it on a list")
			if err != nil {
				return verdict{}, err
			}

			if over {
				return verdict{List: finished, Text: text}, nil
			}
		}
	}

	mine, err := a.choose("Who should do it?", []option{
		{Key: "me", Label: "Me"},
		{Key: "other", Label: "Someone else"},
	})
	if err != nil {
		return verdict{}, err
	}

	if mine == "other" {
		who, err := a.text("Waiting on whom?", "A name or a team", "", true, nil)
		if err != nil {
			return verdict{}, err
		}

		return verdict{List: waiting, Text: text, Who: who}, nil
	}

	dated, err := a.yes("Must it happen on a specific date or time?", "Only for appointments and hard deadlines")
	if err != nil {
		return verdict{}, err
	}

	if dated {
		when, err := a.text("When?", "For example tomorrow, mon 14:00, +3d or 2026-11-20", "", true, valid)
		if err != nil {
			return verdict{}, err
		}

		due, clock, err := moment(when, time.Now())
		if err != nil {
			return verdict{}, err
		}

		return verdict{List: calendar, Text: text, Due: due, Clock: clock}, nil
	}

	context, err := a.text("Context", "Optional, for example @home or @computer, press enter to skip", "", false, nil)
	if err != nil {
		return verdict{}, err
	}

	if context != "" && !strings.HasPrefix(context, "@") {
		context = "@" + context
	}

	return verdict{List: actions, Text: text, Context: context}, nil
}

func (b *book) file(v verdict, id, parent int) *item {
	var target *item

	if id > 0 {
		target = b.find(id)
	}

	if v.Trash {
		if id > 0 {
			b.drop(id)
		}
		return nil
	}

	if target == nil {
		target = b.create(item{})
	}

	was := target.List

	target.Text = v.Text
	target.List = v.List
	target.Who = v.Who
	target.Context = v.Context
	target.Due = v.Due
	target.Clock = v.Clock
	target.Closed = time.Time{}

	if parent > 0 {
		target.Parent = parent
	}

	if v.List == finished {
		target.Closed = time.Now()
	}

	number := target.ID

	if was == projects && v.List != projects {
		for at := range b.Items {
			if b.Items[at].Parent == number {
				b.Items[at].Parent = 0
			}
		}
	}

	if v.Follow != nil {
		b.file(*v.Follow, 0, number)
		return b.find(number)
	}

	return target
}
