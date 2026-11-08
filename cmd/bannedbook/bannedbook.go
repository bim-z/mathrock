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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func sentence(args []string) (string, error) {
	text := strings.Join(strings.Fields(strings.Join(args, " ")), " ")
	if text == "" {
		return "", errors.New("write something to capture")
	}
	return text, nil
}

func summary(b *book, id int) {
	entry := b.find(id)
	if entry == nil {
		fmt.Println("\n  " + faint.Render("trashed") + "\n")
		return
	}

	now := time.Now()

	fmt.Println()

	if entry.List == projects {
		fmt.Println("  " + pink.Render("→") + " " + titles[entry.List] + "  " + faint.Render(entry.Text))

		for _, kid := range b.kids(entry.ID) {
			fmt.Println("  " + pink.Render("→") + " " + titles[kid.List] + "  " + faint.Render(kid.Text))
		}

		fmt.Println()
		return
	}

	detail := entry.Text

	switch entry.List {
	case waiting:
		detail += "  " + entry.Who
	case calendar:
		detail += "  " + when(*entry, now)
	case actions:
		detail += "  " + entry.Context
	}

	fmt.Println("  " + pink.Render("→") + " " + titles[entry.List] + "  " + faint.Render(strings.TrimSpace(detail)))
	fmt.Println()
}

func conclude(b *book, id int, v verdict) error {
	got := b.file(v, id, 0)

	if err := b.save(); err != nil {
		return err
	}

	if got == nil {
		fmt.Println("\n  " + faint.Render("trashed") + "\n")
		return nil
	}

	summary(b, got.ID)

	return nil
}

func capture(args []string, plain bool) error {
	text, err := sentence(args)
	if err != nil {
		return err
	}

	b, err := open()
	if err != nil {
		return err
	}

	entry := b.create(item{Text: text, List: inbox})
	id := entry.ID

	if err := b.save(); err != nil {
		return err
	}

	if plain {
		fmt.Println("\n  " + pink.Render("→") + " " + titles[inbox] + "  " + faint.Render(text) + "\n")
		return nil
	}

	fmt.Println("\n  " + strong.Render(text) + "\n")

	return settle(b, id, newprompt())
}

func settle(b *book, id int, a asker) error {
	entry := b.find(id)

	v, err := clarify(a, entry.Text)
	if errors.Is(err, errAbort) || errors.Is(err, errtty) {
		fmt.Println("  " + faint.Render("kept in the inbox, run 'gtd clarify' when you are ready") + "\n")
		return nil
	}
	if err != nil {
		return err
	}

	return conclude(b, id, v)
}

func process(args []string) error {
	b, err := open()
	if err != nil {
		return err
	}

	a := newprompt()

	if len(args) > 0 {
		number, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("%q is not a number", args[0])
		}

		entry, err := b.pick(number)
		if err != nil {
			return err
		}

		fmt.Println("\n  " + strong.Render(entry.Text) + "\n")

		return settle(b, entry.ID, a)
	}

	var queue []int
	for _, entry := range b.Items {
		if entry.List == inbox {
			queue = append(queue, entry.ID)
		}
	}

	if len(queue) == 0 {
		fmt.Println("\n  " + green.Render("the inbox is empty") + "\n")
		return nil
	}

	for at, id := range queue {
		entry := b.find(id)

		fmt.Println("\n  " + faint.Render(fmt.Sprintf("%d of %d", at+1, len(queue))) + "  " + strong.Render(entry.Text) + "\n")

		v, err := clarify(a, entry.Text)
		if errors.Is(err, errAbort) {
			fmt.Println("  " + faint.Render("stopped, the rest stays in the inbox") + "\n")
			return nil
		}
		if err != nil {
			return err
		}

		if err := conclude(b, id, v); err != nil {
			return err
		}
	}

	return nil
}

func finish(number int) error {
	b, err := open()
	if err != nil {
		return err
	}

	entry, err := b.pick(number)
	if err != nil {
		return err
	}

	id, parent, text := entry.ID, entry.Parent, entry.Text

	entry.List = finished
	entry.Closed = time.Now()

	for at := range b.Items {
		if b.Items[at].Parent == id && b.Items[at].List != finished {
			b.Items[at].List = finished
			b.Items[at].Closed = time.Now()
		}
	}

	if err := b.save(); err != nil {
		return err
	}

	fmt.Println("\n  " + green.Render("done") + "  " + faint.Render(text) + "\n")

	owner := b.find(parent)
	if owner == nil || owner.List != projects || len(b.kids(parent)) > 0 {
		return nil
	}

	a := newprompt()

	step, err := a.text("Next action for "+owner.Text, "Press enter to skip and review it later", "", false, nil)
	if errors.Is(err, errAbort) || errors.Is(err, errtty) || (err == nil && step == "") {
		return nil
	}
	if err != nil {
		return err
	}

	v, err := route(a, step, false)
	if errors.Is(err, errAbort) {
		return nil
	}
	if err != nil {
		return err
	}

	made := b.file(v, 0, parent)

	if err := b.save(); err != nil {
		return err
	}

	summary(b, made.ID)

	return nil
}

func remove(number int, yes bool) error {
	b, err := open()
	if err != nil {
		return err
	}

	entry, err := b.pick(number)
	if err != nil {
		return err
	}

	if !yes {
		return fmt.Errorf("this deletes %q for good, run again with --yes", entry.Text)
	}

	text := entry.Text
	b.drop(entry.ID)

	if err := b.save(); err != nil {
		return err
	}

	fmt.Println("\n  " + faint.Render("deleted  "+text) + "\n")

	return nil
}

func Bannedbook() *cobra.Command {
	var inboxed, yes bool

	root := &cobra.Command{
		Use:          "bannedbook",
		Aliases:      []string{"bb", "gtd"},
		Short:        "Capture tasks and sort them the Getting Things Done way",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			b, err := open()
			if err != nil {
				return err
			}

			fmt.Print(b.board(""))

			return nil
		},
	}

	add := &cobra.Command{
		Use:   "add <task>",
		Short: "Capture a task and clarify it right away",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return capture(args, inboxed)
		},
	}

	add.Flags().BoolVarP(&inboxed, "inbox", "i", false, "only capture it to the inbox, clarify later")

	clarifying := &cobra.Command{
		Use:   "clarify [number]",
		Short: "Process the inbox, or re-sort one item by its number",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return process(args)
		},
	}

	list := &cobra.Command{
		Use:   "ls [list]",
		Short: "Show every list, or just one: inbox, next, waiting, calendar, projects, someday, reference, done",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			only := ""

			if len(args) > 0 {
				found, err := listof(args[0])
				if err != nil {
					return err
				}
				only = found
			}

			b, err := open()
			if err != nil {
				return err
			}

			fmt.Print(b.board(only))

			return nil
		},
	}

	done := &cobra.Command{
		Use:   "done <number>",
		Short: "Mark an item as done",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("%q is not a number", args[0])
			}
			return finish(number)
		},
	}

	rm := &cobra.Command{
		Use:   "rm <number>",
		Short: "Delete an item for good",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("%q is not a number", args[0])
			}
			return remove(number, yes)
		},
	}

	rm.Flags().BoolVarP(&yes, "yes", "y", false, "confirm the deletion")

	review := &cobra.Command{
		Use:   "review",
		Short: "Check stalled projects, old waiting items and the coming week",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			b, err := open()
			if err != nil {
				return err
			}

			fmt.Print(b.review())

			return nil
		},
	}

	root.AddCommand(add, clarifying, list, done, rm, review)

	return root
}
