package license

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	span    = regexp.MustCompile(`^\d{4}(-\d{4})?$`)
	heading = regexp.MustCompile(`(?im)^#{1,6}\s*licen[cs]e`)
	remote  = regexp.MustCompile(`^git@([^:]+):(.+?)(?:\.git)?$`)
)

var names = []string{"license", "license.md", "license.txt", "licence", "licence.md", "licence.txt", "copying", "copying.md", "copying.txt", "unlicense"}

type holder struct {
	Name    string
	Email   string
	Year    string
	Project string
	URL     string
}

func locate(start string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("%s is not a folder", start)
	}

	for dir := abs; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}

		if filepath.Dir(dir) == dir {
			return abs, nil
		}
	}
}

func inside(dir string) bool {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false
	}

	for cur := abs; ; cur = filepath.Dir(cur) {
		if _, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			return true
		}

		if filepath.Dir(cur) == cur {
			return false
		}
	}
}

func config(dir, key string) string {
	out, err := exec.Command("git", "-C", dir, "config", "--get", key).Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

func address(raw string) string {
	raw = strings.TrimSpace(raw)

	if match := remote.FindStringSubmatch(raw); match != nil {
		return "https://" + match[1] + "/" + match[2]
	}

	raw = strings.TrimSuffix(raw, ".git")

	if strings.HasPrefix(raw, "https://") || strings.HasPrefix(raw, "http://") {
		return raw
	}

	return ""
}

func defaults(dir string) holder {
	return holder{
		Name:    config(dir, "user.name"),
		Email:   config(dir, "user.email"),
		Year:    fmt.Sprint(time.Now().Year()),
		Project: filepath.Base(dir),
		URL:     address(config(dir, "remote.origin.url")),
	}
}

func (g grant) text(who holder) string {
	pairs := []string{"[year]", who.Year, "[fullname]", who.Name, "[project]", who.Project}

	if who.Email != "" {
		pairs = append(pairs, "[email]", who.Email)
	}

	if who.URL != "" {
		pairs = append(pairs, "[projecturl]", who.URL)
	}

	out := strings.NewReplacer(pairs...).Replace(g.Body)

	return strings.TrimRight(out, "\n") + "\n"
}

func existing(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		for _, name := range names {
			if strings.EqualFold(entry.Name(), name) {
				return entry.Name()
			}
		}
	}

	return ""
}

func readme(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(entry.Name(), "readme.md") {
			return entry.Name()
		}
	}

	return ""
}

func mentioned(dir string) bool {
	name := readme(dir)
	if name == "" {
		return true
	}

	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return true
	}

	return heading.Match(data)
}

func declare(dir, file string, g grant, who holder) error {
	name := readme(dir)
	if name == "" {
		return errors.New("no README.md found")
	}

	path := filepath.Join(dir, name)

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	body := string(data)
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}

	line := fmt.Sprintf("\n## License\n\nReleased under the %s. See [%s](%s) for details.\n", g.Title, file, file)

	return os.WriteFile(path, []byte(body+line), 0o644)
}

func hints(dir string, g grant) []string {
	var out []string

	for _, entry := range [][2]string{
		{"package.json", `set "license": "` + g.ID + `"`},
		{"composer.json", `set "license": "` + g.ID + `"`},
		{"Cargo.toml", `set license = "` + g.ID + `"`},
		{"pyproject.toml", `set license = "` + g.ID + `"`},
	} {
		if _, err := os.Stat(filepath.Join(dir, entry[0])); err == nil {
			out = append(out, entry[0]+": "+entry[1])
		}
	}

	return out
}

func validate(who holder) error {
	if strings.TrimSpace(who.Name) == "" {
		return errors.New("a copyright holder is needed, pass --name")
	}

	if !span.MatchString(who.Year) {
		return fmt.Errorf("year %q must look like 2026 or 2020-2026", who.Year)
	}

	return nil
}
