/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package deadguy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	dark  = "dark"
	light = "light"
)

type state struct {
	Apps   string
	System string
}

func shell(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var out, bad bytes.Buffer

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout, cmd.Stderr = &out, &bad

	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("%s was not found", name)
		}

		text := strings.TrimSpace(bad.String())
		if text == "" {
			text = err.Error()
		}

		return "", fmt.Errorf("%s failed: %s", name, text)
	}

	return strings.TrimSpace(out.String()), nil
}

func change(name, scope string) error {
	if scope != "all" && scope != "apps" && scope != "system" {
		return fmt.Errorf("unknown scope %q, choose one of: all, apps, system", scope)
	}

	if !split && scope != "all" {
		return errors.New("scope is only supported on Windows")
	}

	before, err := current()
	if err != nil {
		return err
	}

	target := name

	if name == "toggle" {
		base := before.Apps
		if scope == "system" {
			base = before.System
		}

		target = dark
		if base == dark {
			target = light
		}
	}

	if err := assign(target, scope); err != nil {
		return err
	}

	after := before

	if scope != "system" {
		after.Apps = target
	}
	if scope != "apps" {
		after.System = target
	}

	echo(before, after)

	return nil
}

func setting(name, short string) *cobra.Command {
	var scope string

	cmd := &cobra.Command{
		Use:   name,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return change(name, scope)
		},
	}

	if split {
		cmd.Flags().StringVarP(&scope, "scope", "s", "all", "What to change: all, apps or system")
	} else {
		scope = "all"
	}

	return cmd
}

func Mode() *cobra.Command {
	root := &cobra.Command{
		Use:          "mode",
		Short:        "Switch between dark and light appearance",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			now, err := current()
			if err != nil {
				return err
			}

			show(now, engine())

			return nil
		},
	}

	root.AddCommand(
		setting(dark, "Switch to dark appearance"),
		setting(light, "Switch to light appearance"),
		setting("toggle", "Switch to the opposite appearance"),
	)

	return root
}
