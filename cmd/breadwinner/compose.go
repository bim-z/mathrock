/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package breadwinner

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	solo = iota
	list
	rows
	skill
)

type query struct {
	Ask   string
	Hint  string
	Need  bool
	Date  bool
	Check func(string) string
}

type topic struct {
	Title string
	Note  string
	Mode  int
	Base  bool
	Noun  string
	Asks  []query
	Low   int
	High  int
}

type record struct {
	Values  []string
	Bullets []string
}

type answer struct {
	Topic   topic
	Records []record
}

type profile struct {
	Name  string
	Title string
	Email string
	Phone string
	Place string
	Links []string
}

func required(ask, hint string) query {
	return query{Ask: ask, Hint: hint, Need: true}
}

func when(ask, hint string) query {
	return query{Ask: ask, Hint: hint, Need: true, Date: true, Check: moment}
}

func moment(value string) string {
	if !stamp.MatchString(value) || len([]rune(value)) > 20 {
		return "use a date such as Jan 2022, 2022 or Present"
	}
	return ""
}

func mailbox(value string) string {
	if !mailer.MatchString(value) {
		return "that does not look like an email address"
	}
	return ""
}

func number(value string) string {
	if !phone.MatchString(value) {
		return "use digits only, with an optional + and spaces"
	}
	return ""
}

func details() []query {
	return []query{
		{Ask: "Full name", Hint: "as you want it printed on top", Need: true},
		{Ask: "Job title", Hint: "the role you are applying for, for example Backend Engineer", Need: true},
		{Ask: "Email", Hint: "one you check every day", Need: true, Check: mailbox},
		{Ask: "Phone", Hint: "with country code, for example +62 812 3456 7890", Need: true, Check: number},
		{Ask: "Location", Hint: "city and country", Need: true},
	}
}

func catalog() []topic {
	return []topic{
		{Title: "Summary", Note: "two or three sentences that pitch you", Mode: solo, Base: true,
			Asks: []query{required("Summary", "your strongest value in two or three sentences")}},
		{Title: "Experience", Note: "jobs and internships, newest first", Mode: rows, Base: true, Noun: "position", Low: 1, High: 8,
			Asks: []query{required("Role", "for example Software Engineer"), required("Company", ""), required("Location", "city and country"), when("Start", "for example Jan 2022"), when("End", "for example Dec 2024 or Present")}},
		{Title: "Education", Note: "degrees and schools", Mode: rows, Base: true, Noun: "school", High: 3,
			Asks: []query{required("Degree", "for example Bachelor of Science in Computer Science"), required("School", ""), required("Location", "city and country"), when("Start", "for example 2012"), when("End", "for example 2016 or Expected 2027")}},
		{Title: "Skills", Note: "grouped by category", Mode: skill, Base: true, Noun: "category",
			Asks: []query{required("Category", "for example Languages"), required("Skills", "comma separated, for example Go, Python, SQL")}},
		{Title: "Projects", Note: "side projects, open source, case studies", Mode: rows, Noun: "project", Low: 1, High: 5,
			Asks: []query{required("Project name", ""), {Ask: "Link", Hint: "optional, press enter to skip"}, when("Year", "for example 2024")}},
		{Title: "Achievements", Note: "wins and results worth a line each", Mode: list, Noun: "achievement",
			Asks: []query{required("Achievement", "start with a verb and add a number if you can")}},
		{Title: "Certifications", Note: "licenses and courses", Mode: rows, Noun: "certification",
			Asks: []query{required("Certification", ""), required("Issuer", ""), when("Year", "for example 2023")}},
		{Title: "Awards", Note: "honors and scholarships", Mode: list, Noun: "award",
			Asks: []query{required("Award", "name, issuer and year")}},
		{Title: "Volunteering", Note: "community and organization work", Mode: rows, Noun: "role", High: 4,
			Asks: []query{required("Role", ""), required("Organization", ""), required("Location", "city and country"), when("Start", "for example 2020"), when("End", "for example 2022 or Present")}},
		{Title: "Publications", Note: "papers, articles and talks", Mode: rows, Noun: "publication",
			Asks: []query{required("Title", ""), required("Venue", "journal, conference or site"), when("Year", "for example 2023")}},
		{Title: "Languages", Note: "spoken languages", Mode: solo,
			Asks: []query{required("Languages", "for example Indonesian (Native), English (Professional)")}},
		{Title: "Interests", Note: "hobbies, kept short", Mode: solo,
			Asks: []query{required("Interests", "for example Trail running, open source")}},
	}
}

func tidy(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func clip(value string) string {
	return tidy(strings.ReplaceAll(value, "|", "/"))
}

func capital(value string) string {
	runes := []rune(tidy(value))
	if len(runes) > 0 {
		runes[0] = unicode.ToUpper(runes[0])
	}
	return string(runes)
}

func headline(asks []query, rec record) string {
	var parts, dates []string

	for at, ask := range asks {
		if at >= len(rec.Values) {
			break
		}

		value := clip(rec.Values[at])
		if value == "" {
			continue
		}

		if ask.Date {
			dates = append(dates, capital(value))
			continue
		}

		parts = append(parts, value)
	}

	if len(dates) > 0 {
		parts = append(parts, strings.Join(dates, " - "))
	}

	return strings.Join(parts, " | ")
}

func compose(head profile, picks []answer) string {
	var out strings.Builder

	out.WriteString("---\n")
	fmt.Fprintf(&out, "name: %s\n", tidy(head.Name))
	fmt.Fprintf(&out, "title: %s\n", tidy(head.Title))
	fmt.Fprintf(&out, "email: %s\n", tidy(head.Email))
	fmt.Fprintf(&out, "phone: %s\n", tidy(head.Phone))
	fmt.Fprintf(&out, "location: %s\n", tidy(head.Place))

	if len(head.Links) > 0 {
		out.WriteString("links:\n")
		for _, link := range head.Links {
			fmt.Fprintf(&out, "  - %s\n", tidy(link))
		}
	}

	out.WriteString("---\n")

	for _, pick := range picks {
		fmt.Fprintf(&out, "\n## %s\n\n", pick.Topic.Title)

		switch pick.Topic.Mode {
		case solo:
			for _, rec := range pick.Records {
				out.WriteString(tidy(rec.Values[0]) + "\n")
			}
		case list:
			for _, rec := range pick.Records {
				out.WriteString("- " + tidy(rec.Values[0]) + "\n")
			}
		case skill:
			for _, rec := range pick.Records {
				fmt.Fprintf(&out, "- %s: %s\n", tidy(rec.Values[0]), tidy(rec.Values[1]))
			}
		case rows:
			for at, rec := range pick.Records {
				if at > 0 {
					out.WriteString("\n")
				}

				out.WriteString("### " + headline(pick.Topic.Asks, rec) + "\n")

				for _, line := range rec.Bullets {
					out.WriteString("- " + strings.TrimLeft(tidy(line), "-* ") + "\n")
				}
			}
		}
	}

	return out.String()
}
