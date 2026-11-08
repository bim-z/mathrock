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
	"regexp"
	"strings"
)

func plaintext(d doc) string {
	var out []string

	out = append(out, d.Name)
	if d.Title != "" {
		out = append(out, d.Title)
	}
	if items := d.contact(); len(items) > 0 {
		out = append(out, strings.Join(items, " | "))
	}

	for _, sec := range d.shown() {
		out = append(out, "", strings.ToUpper(sec.Title))

		for _, item := range sec.Blocks {
			switch item.Kind {
			case para, plain, note:
				out = append(out, flat(item.Text))
			case bullet:
				out = append(out, strings.Repeat("  ", item.Level)+"- "+flat(item.Text))
			case entry:
				parts := make([]string, 0, len(item.Parts)+1)
				for _, part := range item.Parts {
					parts = append(parts, flat(part))
				}
				if item.Date != "" {
					parts = append(parts, item.Date)
				}
				out = append(out, strings.Join(parts, " | "))
			}
		}
	}

	return strings.Join(out, "\n") + "\n"
}

var (
	mailer = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[A-Za-z]{2,}$`)
	grid   = regexp.MustCompile(`^\s*\|.*\|\s*$`)
	figure = regexp.MustCompile(`!\[[^\]]*\]\(`)
	months = regexp.MustCompile(`(?i)^(jan|feb|mar|apr|may|mei|jun|jul|aug|agu|sep|oct|okt|nov|dec|des)[a-z]*\.?\s+(19|20)\d{2}`)
	slash  = regexp.MustCompile(`^(0?[1-9]|1[0-2])[/.-](19|20)\d{2}`)
	longer = regexp.MustCompile(`^(19|20)\d{2}`)
)

var known = map[string]bool{
	"summary": true, "professional summary": true, "profile": true, "objective": true, "career objective": true,
	"experience": true, "work experience": true, "professional experience": true, "employment history": true, "work history": true,
	"education": true, "skills": true, "technical skills": true, "core competencies": true,
	"projects": true, "personal projects": true, "key projects": true,
	"certifications": true, "certificates": true, "licenses": true, "courses": true, "training": true,
	"achievements": true, "awards": true, "honors": true, "publications": true,
	"languages": true, "volunteer": true, "volunteering": true, "volunteer experience": true, "organizations": true, "leadership": true,
	"interests": true, "hobbies": true, "references": true,
	"ringkasan": true, "profil": true, "pengalaman": true, "pengalaman kerja": true, "pendidikan": true, "keahlian": true,
	"kemampuan": true, "proyek": true, "sertifikasi": true, "sertifikat": true, "penghargaan": true, "prestasi": true,
	"bahasa": true, "organisasi": true, "relawan": true, "minat": true, "hobi": true, "pelatihan": true, "referensi": true,
}

func kind(date string) string {
	for _, part := range strings.Split(date, "-") {
		part = strings.TrimSpace(part)

		switch {
		case months.MatchString(part):
			return "month and year"
		case slash.MatchString(part):
			return "numeric month and year"
		case longer.MatchString(part) && len(part) == 4:
			continue
		}
	}

	return "year"
}

func dated(title string) bool {
	low := strings.ToLower(title)

	for _, key := range []string{"experience", "pengalaman", "employment", "work history", "education", "pendidikan", "volunteer", "relawan"} {
		if strings.Contains(low, key) {
			return true
		}
	}

	return false
}

func issues(d doc, raw string, pages int) []string {
	var out []string

	switch {
	case d.Email == "":
		out = append(out, "no email address found, recruiters and ATS parsers expect one")
	case !mailer.MatchString(d.Email):
		out = append(out, fmt.Sprintf("the email address %q does not look valid", d.Email))
	}

	if d.Phone == "" {
		out = append(out, "no phone number found")
	}
	if d.Place == "" {
		out = append(out, "no location found, add a city and country")
	}

	var core bool

	var mixed bool

	for _, sec := range d.Sections {
		low := strings.ToLower(sec.Title)

		if dated(sec.Title) {
			core = true
		}

		if !known[low] {
			out = append(out, fmt.Sprintf("the section %q may not be recognised, common names are Summary, Experience, Education, Skills and Projects", sec.Title))
		}

		var count, bullets int

		kinds := map[string]bool{}

		for _, item := range sec.Blocks {
			if len([]rune(item.Text)) > 260 {
				out = append(out, fmt.Sprintf("a line in %q is very long, shorter bullets are easier to scan", sec.Title))
			}

			if item.Kind == bullet {
				bullets++
			}

			if item.Kind != entry {
				continue
			}

			count++

			if item.Date == "" && dated(sec.Title) {
				out = append(out, fmt.Sprintf("the entry %q has no dates, ATS timelines rely on them", flat(item.Parts[0])))
			}

			if item.Date != "" {
				kinds[kind(item.Date)] = true
			}
		}

		if len(kinds) > 1 {
			mixed = true
		}

		if count > 0 && bullets == 0 && strings.Contains(low, "experience") {
			out = append(out, fmt.Sprintf("add bullet points describing your achievements under %q", sec.Title))
		}

		if strings.Contains(low, "summary") || strings.Contains(low, "profile") {
			for _, item := range sec.Blocks {
				if len([]rune(item.Text)) > 600 {
					out = append(out, "the summary is long, keep it to two or three sentences")
				}
			}
		}
	}

	if !core {
		out = append(out, "add an Experience or Education section, ATS parsers look for them first")
	}

	if mixed {
		out = append(out, "dates use mixed formats, pick one such as Jan 2022 or 2022 and keep it everywhere")
	}

	for _, line := range strings.Split(raw, "\n") {
		if grid.MatchString(line) && strings.Contains(line, "-") || figure.MatchString(line) {
			out = append(out, "tables and images are not read by many ATS parsers, use plain text instead")
			break
		}
	}

	if pages > 2 {
		out = append(out, fmt.Sprintf("the resume is %d pages long, one or two pages is ideal", pages))
	}

	return out
}
