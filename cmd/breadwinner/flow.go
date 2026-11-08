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
)

var errAbort = errors.New("cancelled")

type prompter interface {
	ask(q query, tag string) (string, error)
	pick(items []topic) ([]int, error)
	more(label string) (bool, error)
}

func gather(p prompter) (string, error) {
	items := catalog()

	chosen, err := p.pick(items)
	if err != nil {
		return "", err
	}

	var head profile

	values := make([]string, 0, 5)

	for _, ask := range details() {
		value, err := p.ask(ask, "Contact")
		if err != nil {
			return "", err
		}

		values = append(values, value)
	}

	head = profile{Name: values[0], Title: values[1], Email: values[2], Phone: values[3], Place: values[4]}

	for count := 1; ; count++ {
		link, err := p.ask(query{Ask: "Link", Hint: "LinkedIn, GitHub or portfolio, press enter to finish"}, fmt.Sprintf("Contact link %d", count))
		if err != nil {
			return "", err
		}

		if link == "" {
			break
		}

		head.Links = append(head.Links, link)
	}

	var picks []answer

	for _, at := range chosen {
		got, err := interview(p, items[at])
		if err != nil {
			return "", err
		}

		picks = append(picks, got)
	}

	return compose(head, picks), nil
}

func interview(p prompter, item topic) (answer, error) {
	out := answer{Topic: item}

	for count := 1; ; count++ {
		tag := item.Title
		if item.Mode != solo {
			tag = fmt.Sprintf("%s %d", item.Title, count)
		}

		var rec record

		for _, ask := range item.Asks {
			value, err := p.ask(ask, tag)
			if err != nil {
				return out, err
			}

			rec.Values = append(rec.Values, value)
		}

		if item.Mode == rows {
			for len(rec.Bullets) < item.High {
				need := len(rec.Bullets) < item.Low
				hint := "start with a verb and add a number if you can"

				if !need {
					hint += ", press enter to finish"
				}

				line, err := p.ask(query{Ask: "Bullet", Hint: hint, Need: need}, tag)
				if err != nil {
					return out, err
				}

				if line == "" {
					break
				}

				rec.Bullets = append(rec.Bullets, line)
			}
		}

		out.Records = append(out.Records, rec)

		if item.Mode == solo {
			return out, nil
		}

		again, err := p.more("Add another " + item.Noun + "?")
		if err != nil {
			return out, err
		}

		if !again {
			return out, nil
		}
	}
}
