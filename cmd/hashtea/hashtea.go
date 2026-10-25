package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/karrick/godirwalk"
	"github.com/spf13/cobra"
)

var (
	title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 2)

	heading = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	amount  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F5A623"))
	muted   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C"))
	kept    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
	danger  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5F87"))
	success = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))

	frame = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(0, 2)
)

type entry struct {
	path string
	size int64
}

type hashed struct {
	entry
	hash string
}

type group struct {
	hash   string
	size   int64
	keep   string
	extras []string
}

func (set group) wasted() int64 {
	return set.size * int64(len(set.extras))
}

func hashtea() (cmd *cobra.Command) {
	cmd = &cobra.Command{
		Use:   "duplicate [path]",
		Short: "Detect and manage duplicate files",
		Long: `Scan a directory and find duplicate files by content (SHA-256).

Files are grouped by size first, and only files sharing a size are hashed.
The first file (alphabetical order) in each group is treated as the original
and is always kept.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := args[0]
			mode, _ := cmd.Flags().GetString("mode")
			minimum, _ := cmd.Flags().GetInt64("min-size")
			yes, _ := cmd.Flags().GetBool("yes")
			workers, _ := cmd.Flags().GetInt("workers")

			if mode != "check" && mode != "purge" {
				return fmt.Errorf("invalid mode %q: use \"check\" or \"purge\"", mode)
			}

			info, err := os.Stat(root)
			if err != nil {
				return err
			}
			if !info.IsDir() {
				return fmt.Errorf("%s is not a directory", root)
			}

			if workers < 1 {
				workers = runtime.NumCPU()
			}

			out := cmd.OutOrStdout()
			fmt.Fprintln(out, title.Render("🔍 Duplicate File Scanner"))
			fmt.Fprintln(out)

			files, err := collect(root, minimum)
			if err != nil {
				return err
			}
			log.Infof("Found %d files to inspect", len(files))

			shortlist := candidates(files)
			log.Infof("%d candidate files share a size, hashing with %d workers...", len(shortlist), workers)

			groups := grouping(hashing(shortlist, workers))
			if len(groups) == 0 {
				fmt.Fprintln(out)
				fmt.Fprintln(out, success.Render("✔ No duplicate files found."))
				return nil
			}

			render(out, groups)
			summary(out, groups, len(files))

			if mode != "purge" {
				fmt.Fprintln(out, muted.Render("Check mode: nothing was deleted. Use --mode purge to remove duplicates."))
				return nil
			}

			if !yes && !confirm(out, "Delete all duplicates listed above? [y/N]: ") {
				fmt.Fprintln(out, muted.Render("Cancelled."))
				return nil
			}

			purge(out, groups)
			return nil
		},
	}

	cmd.Flags().StringP("mode", "m", "check", "Execution mode: check or purge")
	cmd.Flags().Int64("min-size", 1, "Ignore files smaller than this size in bytes")
	cmd.Flags().BoolP("yes", "y", false, "Skip the confirmation prompt when purging")
	cmd.Flags().IntP("workers", "w", 0, "Number of hashing workers (default: CPU count)")
	return cmd
}

func collect(root string, minimum int64) ([]entry, error) {
	var files []entry

	err := godirwalk.Walk(root, &godirwalk.Options{
		Unsorted: true,
		Callback: func(path string, dirent *godirwalk.Dirent) error {
			if !dirent.IsRegular() {
				return nil
			}

			info, err := os.Stat(path)
			if err != nil {
				log.Warnf("Cannot read %s: %v", path, err)
				return nil
			}

			if info.Size() < minimum {
				return nil
			}

			files = append(files, entry{path: path, size: info.Size()})
			return nil
		},
		ErrorCallback: func(path string, err error) godirwalk.ErrorAction {
			log.Errorf("Error accessing %s: %v", path, err)
			return godirwalk.SkipNode
		},
	})

	return files, err
}

func candidates(files []entry) []entry {
	sizes := make(map[int64][]entry)
	for _, file := range files {
		sizes[file.size] = append(sizes[file.size], file)
	}

	var result []entry
	for _, bucket := range sizes {
		if len(bucket) > 1 {
			result = append(result, bucket...)
		}
	}

	return result
}

func hashing(files []entry, workers int) []hashed {
	jobs := make(chan entry)
	results := make(chan hashed)

	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for file := range jobs {
				sum, err := digest(file.path)
				if err != nil {
					log.Errorf("Failed to hash %s: %v", file.path, err)
					continue
				}
				results <- hashed{entry: file, hash: sum}
			}
		}()
	}

	go func() {
		for _, file := range files {
			jobs <- file
		}
		close(jobs)
		wait.Wait()
		close(results)
	}()

	var collected []hashed
	for item := range results {
		collected = append(collected, item)
	}

	return collected
}

func digest(path string) (string, error) {
	desc, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer desc.Close()

	sum := sha256.New()
	if _, err := io.Copy(sum, desc); err != nil {
		return "", err
	}

	return hex.EncodeToString(sum.Sum(nil)), nil
}

func grouping(items []hashed) []group {
	buckets := make(map[string][]hashed)
	for _, item := range items {
		buckets[item.hash] = append(buckets[item.hash], item)
	}

	var groups []group
	for hash, bucket := range buckets {
		if len(bucket) < 2 {
			continue
		}

		sort.Slice(bucket, func(i, j int) bool {
			return bucket[i].path < bucket[j].path
		})

		set := group{hash: hash, size: bucket[0].size, keep: bucket[0].path}
		for _, item := range bucket[1:] {
			set.extras = append(set.extras, item.path)
		}

		groups = append(groups, set)
	}

	sort.Slice(groups, func(i, j int) bool {
		if groups[i].wasted() != groups[j].wasted() {
			return groups[i].wasted() > groups[j].wasted()
		}
		return groups[i].keep < groups[j].keep
	})

	return groups
}

func render(out io.Writer, groups []group) {
	fmt.Fprintln(out)

	for index, set := range groups {
		fmt.Fprintf(out, "%s  %s  %s\n",
			heading.Render(fmt.Sprintf("Group #%d", index+1)),
			amount.Render(readable(set.size)),
			muted.Render("sha256:"+set.hash[:12]),
		)
		fmt.Fprintf(out, "  %s %s\n", kept.Render("✔ ORIGINAL "), set.keep)

		for position, path := range set.extras {
			branch := "├─"
			if position == len(set.extras)-1 {
				branch = "└─"
			}
			fmt.Fprintf(out, "  %s %s %s\n", muted.Render(branch), danger.Render("✘ DUPLICATE"), path)
		}

		fmt.Fprintln(out)
	}
}

func summary(out io.Writer, groups []group, total int) {
	var count int
	var waste int64

	for _, set := range groups {
		count += len(set.extras)
		waste += set.wasted()
	}

	lines := []string{
		heading.Render("Summary"),
		"",
		fmt.Sprintf("Files scanned    : %d", total),
		fmt.Sprintf("Duplicate groups : %d", len(groups)),
		fmt.Sprintf("Duplicate files  : %d", count),
		fmt.Sprintf("Wasted space     : %s", amount.Render(readable(waste))),
	}

	fmt.Fprintln(out, frame.Render(strings.Join(lines, "\n")))
	fmt.Fprintln(out)
}

func confirm(out io.Writer, prompt string) bool {
	fmt.Fprint(out, danger.Render(prompt))

	answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))

	return answer == "y" || answer == "yes"
}

func purge(out io.Writer, groups []group) {
	var removed, failed int
	var freed int64

	for _, set := range groups {
		for _, path := range set.extras {
			if err := os.Remove(path); err != nil {
				log.Errorf("Failed to remove %s: %v", path, err)
				failed++
				continue
			}

			log.Warnf("File purged: %s", path)
			removed++
			freed += set.size
		}
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s %d files removed, %s freed",
		success.Render("✔ Done:"), removed, amount.Render(readable(freed)))

	if failed > 0 {
		fmt.Fprintf(out, ", %s", danger.Render(fmt.Sprintf("%d failed", failed)))
	}

	fmt.Fprintln(out)
}

func readable(bytes int64) string {
	const unit = 1024

	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	divisor, exponent := int64(unit), 0
	for number := bytes / unit; number >= unit; number /= unit {
		divisor *= unit
		exponent++
	}

	return fmt.Sprintf("%.2f %ciB", float64(bytes)/float64(divisor), "KMGTPE"[exponent])
}
