package license

import (
	"embed"
	"fmt"
	"sort"
	"strings"
	"sync"
)

//go:embed texts/*.txt
var texts embed.FS

type grant struct {
	Key   string
	ID    string
	Title string
	Brief string
	Star  bool
	Can   []string
	Must  []string
	Not   []string
	Body  string
}

var (
	cache []grant
	once  sync.Once
)

var shortcuts = map[string]string{
	"apache":   "apache-2.0",
	"gpl":      "gpl-3.0",
	"agpl":     "agpl-3.0",
	"lgpl":     "lgpl-3.0",
	"mpl":      "mpl-2.0",
	"bsd":      "bsd-3-clause",
	"cc0":      "cc0-1.0",
	"epl":      "epl-2.0",
	"eupl":     "eupl-1.2",
	"mit0":     "mit-0",
	"artistic": "artistic-2.0",
	"boost":    "bsl-1.0",
	"ofl":      "ofl-1.1",
}

func humane(tag string) string {
	words := strings.Fields(strings.NewReplacer("--", " ", "-", " ").Replace(tag))
	if len(words) == 0 {
		return tag
	}

	words[0] = strings.ToUpper(words[0][:1]) + words[0][1:]

	return strings.Join(words, " ")
}

func parse(key, raw string) (grant, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")

	parts := strings.SplitN(raw, "\n---\n", 2)
	if len(parts) < 2 || !strings.HasPrefix(parts[0], "---") {
		return grant{}, fmt.Errorf("license %s has no header", key)
	}

	out := grant{Key: key, Body: strings.TrimLeft(parts[1], "\n")}

	var list *[]string

	for _, line := range strings.Split(parts[0], "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "- ") && list != nil {
			*list = append(*list, strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- ")))
			continue
		}

		name, value, ok := strings.Cut(line, ":")
		if !ok || strings.HasPrefix(line, " ") {
			continue
		}

		value = strings.TrimSpace(value)
		list = nil

		switch name {
		case "title":
			out.Title = value
		case "spdx-id":
			out.ID = value
		case "description":
			out.Brief = value
		case "featured":
			out.Star = value == "true"
		case "permissions":
			list = &out.Can
		case "conditions":
			list = &out.Must
		case "limitations":
			list = &out.Not
		}
	}

	if out.Title == "" || out.ID == "" {
		return grant{}, fmt.Errorf("license %s is missing its title", key)
	}

	return out, nil
}

func catalog() []grant {
	once.Do(func() {
		entries, err := texts.ReadDir("texts")
		if err != nil {
			return
		}

		for _, entry := range entries {
			data, err := texts.ReadFile("texts/" + entry.Name())
			if err != nil {
				continue
			}

			key := strings.TrimSuffix(entry.Name(), ".txt")

			item, err := parse(key, string(data))
			if err != nil {
				continue
			}

			cache = append(cache, item)
		}

		sort.SliceStable(cache, func(x, y int) bool {
			if cache[x].Star != cache[y].Star {
				return cache[x].Star
			}
			return strings.ToLower(cache[x].Title) < strings.ToLower(cache[y].Title)
		})
	})

	return cache
}

func find(name string) (grant, error) {
	name = strings.ToLower(strings.TrimSpace(name))

	if full, ok := shortcuts[name]; ok {
		name = full
	}

	for _, item := range catalog() {
		if item.Key == name || strings.ToLower(item.ID) == name || strings.ToLower(item.Title) == name {
			return item, nil
		}
	}

	return grant{}, fmt.Errorf("unknown license %q, run 'license ls' to see them all", name)
}

func (g grant) kind() string {
	var strong, weak, network bool

	for _, tag := range g.Must {
		switch {
		case strings.HasPrefix(tag, "network-use"):
			network = true
		case tag == "same-license--file" || tag == "same-license--library":
			weak = true
		case tag == "same-license":
			strong = true
		}
	}

	switch {
	case network && strong:
		return "network copyleft"
	case strong:
		return "strong copyleft"
	case weak:
		return "weak copyleft"
	case len(g.Must) == 0:
		return "no conditions"
	}

	return "permissive"
}
