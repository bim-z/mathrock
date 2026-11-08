package license

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type options struct {
	Dir    string
	Name   string
	Email  string
	Year   string
	File   string
	Readme bool
	Yes    bool
	Text   bool
}

func target(opt options, g grant) (string, string, holder, error) {
	dir, err := locate(opt.Dir)
	if err != nil {
		return "", "", holder{}, err
	}

	who := defaults(dir)

	if opt.Name != "" {
		who.Name = opt.Name
	}

	if opt.Email != "" {
		who.Email = opt.Email
	}

	if opt.Year != "" {
		who.Year = opt.Year
	}

	file := opt.File

	if file == "" {
		file = "LICENSE"
	}

	return dir, file, who, nil
}

func write(dir, file string, g grant, who holder, opt options) error {
	if err := validate(who); err != nil {
		return err
	}

	if found := existing(dir); found != "" {
		if !opt.Yes {
			return fmt.Errorf("%s already exists, pass --yes to replace it", found)
		}

		if opt.File == "" {
			file = found
		}
	}

	if err := os.WriteFile(filepath.Join(dir, file), []byte(g.text(who)), 0o644); err != nil {
		return err
	}

	fmt.Println("\n  " + green.Render("created") + "  " + filepath.Join(dir, file) + faint.Render("  "+g.Title))

	if opt.Readme {
		switch {
		case readme(dir) == "":
			fmt.Println("  " + faint.Render("no README.md to update"))
		case mentioned(dir):
			fmt.Println("  " + faint.Render("README.md already has a license section"))
		default:
			if err := declare(dir, file, g, who); err != nil {
				return err
			}
			fmt.Println("  " + green.Render("updated") + "  README.md" + faint.Render("  added a License section"))
		}
	}

	for _, line := range hints(dir, g) {
		fmt.Println("  " + faint.Render("hint     "+line))
	}

	if !inside(dir) {
		fmt.Println("  " + amber.Render("note") + "     " + faint.Render("this folder is not a git repository"))
	}

	fmt.Println()

	return nil
}

func add(name string, opt options) error {
	g, err := find(name)
	if err != nil {
		return err
	}

	dir, file, who, err := target(opt, g)
	if err != nil {
		return err
	}

	if who.Name == "" && interactive() {
		who.Name, err = text("Copyright holder", "Your name or your organization", "", nil)
		if errors.Is(err, errAbort) {
			return nil
		}
		if err != nil {
			return err
		}
	}

	return write(dir, file, g, who, opt)
}

func wizard(opt options) error {
	if !interactive() {
		return errors.New("the guided mode needs an interactive terminal, use: license add <name>")
	}

	dir, file, who, err := target(opt, grant{})
	if err != nil {
		return err
	}

	if found := existing(dir); found != "" && !opt.Yes {
		again, err := yes(found+" already exists, replace it?", "The current file is overwritten", false)
		if errors.Is(err, errAbort) || (err == nil && !again) {
			return nil
		}
		if err != nil {
			return err
		}

		opt.Yes = true
	}

	picks := []option{{Key: "help", Label: "Help me choose"}}
	for _, item := range catalog() {
		picks = append(picks, option{Key: item.Key, Label: item.Title})
	}

	pick, err := choose("Which license?", "Use the arrows, press enter to select", picks, 12)
	if errors.Is(err, errAbort) {
		return nil
	}
	if err != nil {
		return err
	}

	if pick == "help" {
		pick, err = advise()
		if errors.Is(err, errAbort) {
			return nil
		}
		if err != nil {
			return err
		}
	}

	g, err := find(pick)
	if err != nil {
		return err
	}

	fmt.Print(summary(g))

	who.Name, err = text("Copyright holder", "Your name or your organization", who.Name, nil)
	if errors.Is(err, errAbort) {
		return nil
	}
	if err != nil {
		return err
	}

	who.Year, err = text("Year", "For example 2026 or 2020-2026", who.Year, func(value string) error {
		if !span.MatchString(value) {
			return errors.New("use a year like 2026 or a range like 2020-2026")
		}
		return nil
	})
	if errors.Is(err, errAbort) {
		return nil
	}
	if err != nil {
		return err
	}

	if !mentioned(dir) {
		opt.Readme, err = yes("Mention the license in README.md?", "Adds a License section at the end", true)
		if errors.Is(err, errAbort) {
			return nil
		}
		if err != nil {
			return err
		}
	}

	return write(dir, file, g, who, opt)
}

func show(name string, opt options) error {
	g, err := find(name)
	if err != nil {
		return err
	}

	if opt.Text {
		who := holder{Name: "[fullname]", Year: "[year]", Project: "[project]"}
		fmt.Print(g.text(who))
		return nil
	}

	fmt.Print(summary(g))

	return nil
}

func Cagematch() *cobra.Command {
	var opt options

	root := &cobra.Command{
		Use:          "cagematch",
		Aliases:      []string{"cagematch"},
		Short:        "Add an open source license to a repository",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return wizard(opt)
		},
	}

	root.Flags().StringVarP(&opt.Dir, "dir", "d", ".", "repository folder")
	root.Flags().BoolVarP(&opt.Yes, "yes", "y", false, "replace an existing license file")

	adding := &cobra.Command{
		Use:   "add <license>",
		Short: "Write a LICENSE file, for example: license add mit",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return add(args[0], opt)
		},
	}

	adding.Flags().StringVarP(&opt.Dir, "dir", "d", ".", "repository folder")
	adding.Flags().StringVarP(&opt.Name, "name", "n", "", "copyright holder, defaults to git user.name")
	adding.Flags().StringVarP(&opt.Email, "email", "e", "", "contact email, defaults to git user.email")
	adding.Flags().StringVar(&opt.Year, "year", "", "year or range, defaults to the current year")
	adding.Flags().StringVarP(&opt.File, "file", "f", "", "file name, defaults to LICENSE")
	adding.Flags().BoolVar(&opt.Readme, "readme", false, "also add a License section to README.md")
	adding.Flags().BoolVarP(&opt.Yes, "yes", "y", false, "replace an existing license file")

	ls := &cobra.Command{
		Use:   "ls [search]",
		Short: "List the available licenses",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			term := ""
			if len(args) > 0 {
				term = args[0]
			}
			fmt.Print(listing(term))
			return nil
		},
	}

	showing := &cobra.Command{
		Use:   "show <license>",
		Short: "Explain a license, or print its full text with --text",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return show(args[0], opt)
		},
	}

	showing.Flags().BoolVar(&opt.Text, "text", false, "print the full license text")

	choosing := &cobra.Command{
		Use:   "choose",
		Short: "Answer a few questions and get a recommendation",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if !interactive() {
				return errors.New("choosing needs an interactive terminal")
			}

			key, err := advise()
			if errors.Is(err, errAbort) {
				return nil
			}
			if err != nil {
				return err
			}

			g, err := find(key)
			if err != nil {
				return err
			}

			fmt.Print(summary(g))
			fmt.Println("  " + faint.Render("add it with: license add "+g.Key) + "\n")

			return nil
		},
	}

	root.AddCommand(adding, ls, showing, choosing)

	return root
}
