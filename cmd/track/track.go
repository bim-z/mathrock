/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package track

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

type options struct {
	Project string
	Last    int
	Date    string
	Yes     bool
}

func stamp(date string) (time.Time, error) {
	now := time.Now()
	if strings.TrimSpace(date) == "" {
		return now, nil
	}

	var day time.Time

	switch strings.ToLower(strings.TrimSpace(date)) {
	case "today":
		day = now
	case "yesterday":
		day = now.AddDate(0, 0, -1)
	default:
		parsed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(date), time.Local)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid date %q, use YYYY-MM-DD, today or yesterday", date)
		}
		day = parsed
	}

	when := time.Date(day.Year(), day.Month(), day.Day(), now.Hour(), now.Minute(), now.Second(), 0, time.Local)
	if when.After(now) {
		return time.Time{}, errors.New("the date is in the future")
	}

	return when, nil
}

func numbers(args []string, total int) ([]int, error) {
	seen := map[int]bool{}
	var out []int

	for _, arg := range args {
		num, err := strconv.Atoi(arg)
		if err != nil {
			return nil, fmt.Errorf("%q is not a number", arg)
		}

		if num < 1 || num > total {
			if total == 0 {
				return nil, errors.New("the project has no checkpoints")
			}
			return nil, fmt.Errorf("no checkpoint %d, the project has %s", num, plural(total, "checkpoint"))
		}

		if !seen[num] {
			seen[num] = true
			out = append(out, num)
		}
	}

	return out, nil
}

func fresh(args []string) error {
	name, err := clean(strings.Join(args, " "))
	if err != nil {
		return err
	}

	err = update(func(tx *bbolt.Tx) error {
		if tx.Bucket(projects).Get([]byte(slug(name))) != nil {
			return fmt.Errorf("project %q already exists", name)
		}

		if err := keep(tx, project{Name: name, Born: time.Now()}); err != nil {
			return err
		}

		return tx.Bucket(meta).Put(active, []byte(slug(name)))
	})
	if err != nil {
		return err
	}

	fmt.Println("\n  " + pink.Render("created") + " " + name + faint.Render("  add a checkpoint with 'track add'") + "\n")

	return nil
}

func use(args []string) error {
	name := strings.Join(args, " ")

	var found project

	err := update(func(tx *bbolt.Tx) error {
		item, err := lookup(tx, name)
		if err != nil {
			return err
		}

		found = item

		return tx.Bucket(meta).Put(active, []byte(slug(item.Name)))
	})
	if err != nil {
		return err
	}

	fmt.Println("\n  " + faint.Render("using") + " " + found.Name + "\n")

	return nil
}

func add(args []string, opt options) error {
	text := strings.Join(strings.Fields(strings.Join(args, " ")), " ")
	if text == "" {
		return errors.New("checkpoint text is empty")
	}

	when, err := stamp(opt.Date)
	if err != nil {
		return err
	}

	var proj project
	var items []point

	err = update(func(tx *bbolt.Tx) error {
		var err error

		proj, err = chosen(tx, opt.Project)
		if err != nil {
			return err
		}

		seq, err := tx.Bucket(points).NextSequence()
		if err != nil {
			return fmt.Errorf("failed to create checkpoint: %w", err)
		}

		if err := store(tx, point{ID: seq, Project: slug(proj.Name), Text: text, Born: when}); err != nil {
			return err
		}

		items, err = collect(tx, proj)

		return err
	})
	if err != nil {
		return err
	}

	fmt.Print(timeline(proj, items, 3, width(), time.Now()))

	return nil
}

func show(name string, opt options) error {
	var proj project
	var items []point

	err := view(func(tx *bbolt.Tx) error {
		var err error

		if name == "" {
			name = opt.Project
		}

		proj, err = chosen(tx, name)
		if err != nil {
			return err
		}

		items, err = collect(tx, proj)

		return err
	})
	if err != nil {
		return err
	}

	fmt.Print(timeline(proj, items, opt.Last, width(), time.Now()))

	return nil
}

func edit(args []string, opt options) error {
	text := strings.Join(strings.Fields(strings.Join(args[1:], " ")), " ")
	if text == "" {
		return errors.New("checkpoint text is empty")
	}

	var proj project
	var items []point

	err := update(func(tx *bbolt.Tx) error {
		var err error

		proj, err = chosen(tx, opt.Project)
		if err != nil {
			return err
		}

		items, err = collect(tx, proj)
		if err != nil {
			return err
		}

		pick, err := numbers(args[:1], len(items))
		if err != nil {
			return err
		}

		target := items[pick[0]-1]
		target.Text = text
		items[pick[0]-1] = target

		return store(tx, target)
	})
	if err != nil {
		return err
	}

	fmt.Print(timeline(proj, items, opt.Last, width(), time.Now()))

	return nil
}

func remove(args []string, opt options) error {
	var gone []string

	err := update(func(tx *bbolt.Tx) error {
		proj, err := chosen(tx, opt.Project)
		if err != nil {
			return err
		}

		items, err := collect(tx, proj)
		if err != nil {
			return err
		}

		pick, err := numbers(args, len(items))
		if err != nil {
			return err
		}

		for _, num := range pick {
			target := items[num-1]

			if err := tx.Bucket(points).Delete(key(target.ID)); err != nil {
				return fmt.Errorf("failed to remove checkpoint: %w", err)
			}

			gone = append(gone, faint.Render(fmt.Sprintf("%d", num))+"  "+target.Text)
		}

		return nil
	})
	if err != nil {
		return err
	}

	fmt.Println("\n  " + faint.Render("removed") + "\n\n  " + strings.Join(gone, "\n  ") + "\n")

	return nil
}

func list() error {
	var names []project
	var current string

	counts := map[string]int{}
	latest := map[string]time.Time{}

	err := view(func(tx *bbolt.Tx) error {
		var err error

		names, err = every(tx)
		if err != nil {
			return err
		}

		all, err := entries(tx)
		if err != nil {
			return err
		}

		for _, item := range all {
			counts[item.Project]++
			if item.Born.After(latest[item.Project]) {
				latest[item.Project] = item.Born
			}
		}

		if raw := tx.Bucket(meta).Get(active); raw != nil {
			current = string(raw)
		}

		return nil
	})
	if err != nil {
		return err
	}

	fmt.Print(summary(time.Now(), names, counts, latest, current))

	return nil
}

func rename(args []string) error {
	name, err := clean(args[1])
	if err != nil {
		return err
	}

	err = update(func(tx *bbolt.Tx) error {
		old, err := lookup(tx, args[0])
		if err != nil {
			return err
		}

		if slug(name) != slug(old.Name) && tx.Bucket(projects).Get([]byte(slug(name))) != nil {
			return fmt.Errorf("project %q already exists", name)
		}

		items, err := collect(tx, old)
		if err != nil {
			return err
		}

		if err := tx.Bucket(projects).Delete([]byte(slug(old.Name))); err != nil {
			return fmt.Errorf("failed to rename project: %w", err)
		}

		old.Name = name
		if err := keep(tx, old); err != nil {
			return err
		}

		for _, item := range items {
			item.Project = slug(name)
			if err := store(tx, item); err != nil {
				return err
			}
		}

		if raw := tx.Bucket(meta).Get(active); raw != nil && string(raw) == slug(args[0]) {
			return tx.Bucket(meta).Put(active, []byte(slug(name)))
		}

		return nil
	})
	if err != nil {
		return err
	}

	fmt.Println("\n  " + faint.Render("renamed to") + " " + name + "\n")

	return nil
}

func drop(name string, opt options) error {
	if !opt.Yes {
		return errors.New("this removes the project and all of its checkpoints, use --yes to confirm")
	}

	var count int

	err := update(func(tx *bbolt.Tx) error {
		old, err := lookup(tx, name)
		if err != nil {
			return err
		}

		items, err := collect(tx, old)
		if err != nil {
			return err
		}

		for _, item := range items {
			if err := tx.Bucket(points).Delete(key(item.ID)); err != nil {
				return fmt.Errorf("failed to remove checkpoint: %w", err)
			}
		}

		count = len(items)

		if err := tx.Bucket(projects).Delete([]byte(slug(old.Name))); err != nil {
			return fmt.Errorf("failed to remove project: %w", err)
		}

		if raw := tx.Bucket(meta).Get(active); raw != nil && string(raw) == slug(old.Name) {
			return tx.Bucket(meta).Delete(active)
		}

		return nil
	})
	if err != nil {
		return err
	}

	fmt.Println("\n  " + faint.Render("dropped") + " " + name + faint.Render("  with "+plural(count, "checkpoint")) + "\n")

	return nil
}

func Track() *cobra.Command {
	var opt options

	root := &cobra.Command{
		Use:          "track",
		Short:        "Keep a timeline of the changes in your projects",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return show("", opt)
		},
	}

	root.PersistentFlags().StringVarP(&opt.Project, "project", "p", "", "Project to use instead of the active one")
	root.Flags().IntVarP(&opt.Last, "last", "n", 0, "Only show the latest checkpoints")

	showing := &cobra.Command{
		Use:   "show [project]",
		Short: "Show the timeline of a project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			return show(name, opt)
		},
	}
	showing.Flags().IntVarP(&opt.Last, "last", "n", 0, "Only show the latest checkpoints")

	adding := &cobra.Command{
		Use:   "add <text>",
		Short: "Add a checkpoint to the active project",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return add(args, opt)
		},
	}
	adding.Flags().StringVarP(&opt.Date, "date", "d", "", "Date of the checkpoint: YYYY-MM-DD, today or yesterday")

	dropping := &cobra.Command{
		Use:   "drop <project>",
		Short: "Delete a project and all of its checkpoints",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return drop(strings.Join(args, " "), opt)
		},
	}
	dropping.Flags().BoolVar(&opt.Yes, "yes", false, "Confirm the deletion")

	root.AddCommand(
		&cobra.Command{
			Use:   "new <project>",
			Short: "Create a project and make it active",
			Args:  cobra.MinimumNArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				return fresh(args)
			},
		},
		&cobra.Command{
			Use:   "use <project>",
			Short: "Make a project active",
			Args:  cobra.MinimumNArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				return use(args)
			},
		},
		adding,
		showing,
		&cobra.Command{
			Use:   "ls",
			Short: "List all projects",
			Args:  cobra.NoArgs,
			RunE: func(_ *cobra.Command, _ []string) error {
				return list()
			},
		},
		&cobra.Command{
			Use:   "edit <number> <text>",
			Short: "Change the text of a checkpoint",
			Args:  cobra.MinimumNArgs(2),
			RunE: func(_ *cobra.Command, args []string) error {
				return edit(args, opt)
			},
		},
		&cobra.Command{
			Use:   "rm <number>...",
			Short: "Remove checkpoints by number",
			Args:  cobra.MinimumNArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				return remove(args, opt)
			},
		},
		&cobra.Command{
			Use:   "rename <project> <new name>",
			Short: "Rename a project",
			Args:  cobra.ExactArgs(2),
			RunE: func(_ *cobra.Command, args []string) error {
				return rename(args)
			},
		},
		dropping,
		tui(&opt),
	)

	return root
}
