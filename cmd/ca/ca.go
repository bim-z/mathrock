package calendar

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

const layout = "2006-01-02"

var bucket = []byte("calendar")

type event struct {
	ID    uint64 `json:"id"`
	Title string `json:"title"`
	Day   string `json:"day"`
	Begin int    `json:"begin"`
	End   int    `json:"end"`
	Place string `json:"place"`
}

type choose func([]event) ([]event, error)

func key(id uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, id)
	return buf
}

func midnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

func parse(stamp string) time.Time {
	out, err := time.ParseInLocation(layout, stamp, time.Local)
	if err != nil {
		return time.Time{}
	}
	return out
}

func shift(t time.Time, months int) time.Time {
	first := time.Date(t.Year(), t.Month()+time.Month(months), 1, 0, 0, 0, 0, time.Local)
	last := time.Date(first.Year(), first.Month()+1, 0, 0, 0, 0, 0, time.Local).Day()

	num := t.Day()
	if num > last {
		num = last
	}

	return time.Date(first.Year(), first.Month(), num, 0, 0, 0, 0, time.Local)
}

func when(text string, now time.Time) (time.Time, error) {
	text = strings.ToLower(strings.TrimSpace(text))
	today := midnight(now)

	switch text {
	case "", "today":
		return today, nil
	case "tomorrow":
		return today.AddDate(0, 0, 1), nil
	case "yesterday":
		return today.AddDate(0, 0, -1), nil
	}

	if text[0] == '+' || text[0] == '-' {
		if num, err := strconv.Atoi(text); err == nil {
			return today.AddDate(0, 0, num), nil
		}
	}

	for i := 0; i < 7; i++ {
		full := strings.ToLower(time.Weekday(i).String())
		if text == full || text == full[:3] {
			ahead := (i - int(today.Weekday()) + 7) % 7
			if ahead == 0 {
				ahead = 7
			}
			return today.AddDate(0, 0, ahead), nil
		}
	}

	if date, err := time.ParseInLocation(layout, text, time.Local); err == nil {
		return date, nil
	}

	if date, err := time.ParseInLocation("02/01/2006", text, time.Local); err == nil {
		return date, nil
	}

	if date, err := time.ParseInLocation("02/01", text, time.Local); err == nil {
		return time.Date(today.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local), nil
	}

	return time.Time{}, fmt.Errorf("invalid date %q", text)
}

func clock(text string) (int, error) {
	raw := strings.ToLower(strings.TrimSpace(text))
	pm := strings.HasSuffix(raw, "pm")
	am := strings.HasSuffix(raw, "am")
	raw = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(raw, "pm"), "am"))

	parts := strings.Split(raw, ":")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid time %q", text)
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid time %q", text)
	}

	var minute int
	if len(parts) == 2 {
		minute, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("invalid time %q", text)
		}
	}

	if am || pm {
		if hour < 1 || hour > 12 {
			return 0, fmt.Errorf("invalid time %q", text)
		}
		hour %= 12
		if pm {
			hour += 12
		}
	}

	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, fmt.Errorf("invalid time %q", text)
	}

	return hour*60 + minute, nil
}

func span(text string) (int, int, error) {
	text = strings.TrimSpace(text)
	if text == "" || strings.EqualFold(text, "all") {
		return -1, -1, nil
	}

	left, right, found := strings.Cut(text, "-")

	begin, err := clock(left)
	if err != nil {
		return 0, 0, err
	}
	if !found {
		return begin, -1, nil
	}

	end, err := clock(right)
	if err != nil {
		return 0, 0, err
	}
	if end <= begin {
		return 0, 0, errors.New("end must be after start")
	}

	return begin, end, nil
}

func hhmm(minutes int) string {
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}

func slot(ev event) string {
	if ev.Begin < 0 {
		return ""
	}
	if ev.End < 0 {
		return hhmm(ev.Begin)
	}
	return hhmm(ev.Begin) + "-" + hhmm(ev.End)
}

func label(ev event) string {
	if text := slot(ev); text != "" {
		return text
	}
	return "all day"
}

func anchor(args []string) (time.Time, error) {
	if len(args) == 0 {
		return midnight(time.Now()), nil
	}
	return when(args[0], time.Now())
}

func open() (*bbolt.DB, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to locate config directory: %w", err)
	}

	dir = filepath.Join(dir, "mathrock")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := bbolt.Open(filepath.Join(dir, "calendar.db"), 0o600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucket)
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to prepare database: %w", err)
	}

	return db, nil
}

func update(fn func(*bbolt.Bucket) error) error {
	db, err := open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bbolt.Tx) error {
		return fn(tx.Bucket(bucket))
	})
}

func view(fn func(*bbolt.Bucket) error) error {
	db, err := open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.View(func(tx *bbolt.Tx) error {
		return fn(tx.Bucket(bucket))
	})
}

func load(b *bbolt.Bucket) ([]event, error) {
	var out []event

	err := b.ForEach(func(_, v []byte) error {
		var ev event
		if err := json.Unmarshal(v, &ev); err != nil {
			return fmt.Errorf("failed to decode event: %w", err)
		}
		out = append(out, ev)
		return nil
	})

	return out, err
}

func save(b *bbolt.Bucket, ev event) error {
	raw, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("failed to encode event: %w", err)
	}

	if err := b.Put(key(ev.ID), raw); err != nil {
		return fmt.Errorf("failed to save event: %w", err)
	}

	return nil
}

func order(in []event) []event {
	out := make([]event, len(in))
	copy(out, in)

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Day != out[j].Day {
			return out[i].Day < out[j].Day
		}
		if out[i].Begin != out[j].Begin {
			return out[i].Begin < out[j].Begin
		}
		return out[i].ID < out[j].ID
	})

	return out
}

func fetch() ([]event, error) {
	var out []event

	err := view(func(b *bbolt.Bucket) error {
		all, err := load(b)
		out = order(all)
		return err
	})

	return out, err
}

func numbers(args []string) choose {
	return func(all []event) ([]event, error) {
		var out []event
		seen := map[int]bool{}

		for _, arg := range args {
			num, err := strconv.Atoi(arg)
			if err != nil || num < 1 || num > len(all) {
				return nil, fmt.Errorf("no event numbered %q", arg)
			}
			if seen[num] {
				continue
			}
			seen[num] = true
			out = append(out, all[num-1])
		}

		return out, nil
	}
}

func ident(id uint64) choose {
	return func(all []event) ([]event, error) {
		for _, ev := range all {
			if ev.ID == id {
				return []event{ev}, nil
			}
		}
		return nil, errors.New("event no longer exists")
	}
}

func change(sel choose, fn func(*event)) ([]event, error) {
	var out []event

	err := update(func(b *bbolt.Bucket) error {
		all, err := load(b)
		if err != nil {
			return err
		}

		hit, err := sel(order(all))
		if err != nil {
			return err
		}

		for _, ev := range hit {
			fn(&ev)
			if err := save(b, ev); err != nil {
				return err
			}
			out = append(out, ev)
		}

		return nil
	})

	return out, err
}

func remove(sel choose) ([]event, error) {
	var hit []event

	err := update(func(b *bbolt.Bucket) error {
		all, err := load(b)
		if err != nil {
			return err
		}

		hit, err = sel(order(all))
		if err != nil {
			return err
		}

		for _, ev := range hit {
			if err := b.Delete(key(ev.ID)); err != nil {
				return fmt.Errorf("failed to delete event: %w", err)
			}
		}

		return nil
	})

	return hit, err
}

func insert(fresh event) (event, int, error) {
	var num int

	err := update(func(b *bbolt.Bucket) error {
		id, err := b.NextSequence()
		if err != nil {
			return fmt.Errorf("failed to allocate id: %w", err)
		}

		fresh.ID = id
		if err := save(b, fresh); err != nil {
			return err
		}

		all, err := load(b)
		if err != nil {
			return err
		}

		for i, cur := range order(all) {
			if cur.ID == id {
				num = i + 1
			}
		}

		return nil
	})

	return fresh, num, err
}

func Calendar() *cobra.Command {
	root := &cobra.Command{
		Use:          "calendar",
		Short:        "Keep track of events on a calendar",
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return dash()
		},
	}

	root.AddCommand(month(), week(), day(), ls(), add(), edit(), rm(), find(), prune(), tui())

	return root
}

func month() *cobra.Command {
	return &cobra.Command{
		Use:   "month [date]",
		Short: "Show a month with its events",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			at, err := anchor(args)
			if err != nil {
				return err
			}

			all, err := fetch()
			if err != nil {
				return err
			}

			prefix := at.Format("2006-01")
			lines := listing(all, func(ev event) bool { return strings.HasPrefix(ev.Day, prefix) })

			fmt.Println("\n" + grid(at, time.Time{}, marks(all), time.Now()) + "\n")
			if len(lines) > 0 {
				fmt.Println(strings.Join(lines, "\n") + "\n")
			}

			return nil
		},
	}
}

func week() *cobra.Command {
	return &cobra.Command{
		Use:   "week [date]",
		Short: "Show the week of a date, starting on Monday",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			at, err := anchor(args)
			if err != nil {
				return err
			}

			all, err := fetch()
			if err != nil {
				return err
			}

			start := at.AddDate(0, 0, -((int(at.Weekday()) + 6) % 7))

			var lines []string
			for i := 0; i < 7; i++ {
				lines = append(lines, block(start.AddDate(0, 0, i), all, true)...)
				lines = append(lines, "")
			}

			fmt.Println("\n" + strings.Join(lines, "\n"))
			return nil
		},
	}
}

func day() *cobra.Command {
	return &cobra.Command{
		Use:   "day [date]",
		Short: "Show a single day",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			at, err := anchor(args)
			if err != nil {
				return err
			}

			all, err := fetch()
			if err != nil {
				return err
			}

			fmt.Println("\n" + strings.Join(block(at, all, true), "\n") + "\n")
			return nil
		},
	}
}

func ls() *cobra.Command {
	return &cobra.Command{
		Use:   "ls [days]",
		Short: "List upcoming events, 14 days by default",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			ahead := 14
			if len(args) == 1 {
				num, err := strconv.Atoi(args[0])
				if err != nil || num < 1 {
					return fmt.Errorf("days must be a positive number, got %q", args[0])
				}
				ahead = num
			}

			all, err := fetch()
			if err != nil {
				return err
			}

			lines := upcoming(all, ahead)
			if len(lines) == 0 {
				fmt.Println(faint.Render(fmt.Sprintf("  no events in the next %d days", ahead)))
				return nil
			}

			fmt.Println("\n" + strings.Join(lines, "\n") + "\n")
			return nil
		},
	}
}

func add() *cobra.Command {
	var on, at, place string

	cmd := &cobra.Command{
		Use:   "add <title>",
		Short: "Add an event",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := strings.TrimSpace(strings.Join(args, " "))
			if name == "" {
				return errors.New("title is empty")
			}

			date, err := when(on, time.Now())
			if err != nil {
				return err
			}

			begin, end, err := span(at)
			if err != nil {
				return err
			}

			ev, num, err := insert(event{
				Title: name,
				Day:   date.Format(layout),
				Begin: begin,
				End:   end,
				Place: strings.TrimSpace(place),
			})
			if err != nil {
				return err
			}

			fmt.Println(faint.Render("  added ") + pink.Render(strconv.Itoa(num)) + " " + ev.Title + faint.Render("  "+date.Format("Mon, 02 Jan")+"  "+label(ev)))
			return nil
		},
	}

	cmd.Flags().StringVarP(&on, "on", "o", "today", "date: today, tomorrow, mon, +3, 2026-12-31 or 31/12")
	cmd.Flags().StringVarP(&at, "at", "a", "", "time: 14:00, 14:00-15:30 or 2pm, empty for all day")
	cmd.Flags().StringVarP(&place, "place", "p", "", "where it happens")

	return cmd
}

func edit() *cobra.Command {
	var on, at, place string

	cmd := &cobra.Command{
		Use:   "edit <number> [title]",
		Short: "Change an event",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			flags := c.Flags()
			name := strings.TrimSpace(strings.Join(args[1:], " "))

			if name == "" && !flags.Changed("on") && !flags.Changed("at") && !flags.Changed("place") {
				return errors.New("nothing to change")
			}

			var date time.Time
			var begin, end int
			var err error

			if flags.Changed("on") {
				date, err = when(on, time.Now())
				if err != nil {
					return err
				}
			}

			if flags.Changed("at") {
				begin, end, err = span(at)
				if err != nil {
					return err
				}
			}

			hit, err := change(numbers(args[:1]), func(ev *event) {
				if name != "" {
					ev.Title = name
				}
				if flags.Changed("on") {
					ev.Day = date.Format(layout)
				}
				if flags.Changed("at") {
					ev.Begin, ev.End = begin, end
				}
				if flags.Changed("place") {
					ev.Place = strings.TrimSpace(place)
				}
			})
			if err != nil {
				return err
			}

			return echo(hit)
		},
	}

	cmd.Flags().StringVarP(&on, "on", "o", "", "new date")
	cmd.Flags().StringVarP(&at, "at", "a", "", "new time, empty for all day")
	cmd.Flags().StringVarP(&place, "place", "p", "", "new place")

	return cmd
}

func rm() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <number>...",
		Short: "Remove events",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			hit, err := remove(numbers(args))
			if err != nil {
				return err
			}

			for _, ev := range hit {
				fmt.Println(faint.Render("  removed ") + ev.Title + faint.Render("  "+parse(ev.Day).Format("Mon, 02 Jan 2006")))
			}
			return nil
		},
	}
}

func find() *cobra.Command {
	return &cobra.Command{
		Use:   "find <query>",
		Short: "Search events by title or place",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			query := strings.ToLower(strings.Join(args, " "))

			all, err := fetch()
			if err != nil {
				return err
			}

			lines := listing(all, func(ev event) bool {
				return strings.Contains(strings.ToLower(ev.Title), query) || strings.Contains(strings.ToLower(ev.Place), query)
			})
			if len(lines) == 0 {
				fmt.Println(faint.Render("  no events match " + query))
				return nil
			}

			fmt.Println("\n" + strings.Join(lines, "\n") + "\n")
			return nil
		},
	}
}

func prune() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove every event before today",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if !yes {
				return errors.New("pruning requires the --yes flag")
			}

			var count int
			today := midnight(time.Now()).Format(layout)

			err := update(func(b *bbolt.Bucket) error {
				all, err := load(b)
				if err != nil {
					return err
				}

				for _, ev := range all {
					if ev.Day >= today {
						continue
					}
					if err := b.Delete(key(ev.ID)); err != nil {
						return fmt.Errorf("failed to delete event: %w", err)
					}
					count++
				}

				return nil
			})
			if err != nil {
				return err
			}

			fmt.Println(faint.Render("  pruned ") + strconv.Itoa(count) + faint.Render(" past events"))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "confirm the deletion")

	return cmd
}package calendar

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

const layout = "2006-01-02"

var bucket = []byte("calendar")

type event struct {
	ID    uint64 `json:"id"`
	Title string `json:"title"`
	Day   string `json:"day"`
	Begin int    `json:"begin"`
	End   int    `json:"end"`
	Place string `json:"place"`
}

type choose func([]event) ([]event, error)

func key(id uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, id)
	return buf
}

func midnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

func parse(stamp string) time.Time {
	out, err := time.ParseInLocation(layout, stamp, time.Local)
	if err != nil {
		return time.Time{}
	}
	return out
}

func shift(t time.Time, months int) time.Time {
	first := time.Date(t.Year(), t.Month()+time.Month(months), 1, 0, 0, 0, 0, time.Local)
	last := time.Date(first.Year(), first.Month()+1, 0, 0, 0, 0, 0, time.Local).Day()

	num := t.Day()
	if num > last {
		num = last
	}

	return time.Date(first.Year(), first.Month(), num, 0, 0, 0, 0, time.Local)
}

func when(text string, now time.Time) (time.Time, error) {
	text = strings.ToLower(strings.TrimSpace(text))
	today := midnight(now)

	switch text {
	case "", "today":
		return today, nil
	case "tomorrow":
		return today.AddDate(0, 0, 1), nil
	case "yesterday":
		return today.AddDate(0, 0, -1), nil
	}

	if text[0] == '+' || text[0] == '-' {
		if num, err := strconv.Atoi(text); err == nil {
			return today.AddDate(0, 0, num), nil
		}
	}

	for i := 0; i < 7; i++ {
		full := strings.ToLower(time.Weekday(i).String())
		if text == full || text == full[:3] {
			ahead := (i - int(today.Weekday()) + 7) % 7
			if ahead == 0 {
				ahead = 7
			}
			return today.AddDate(0, 0, ahead), nil
		}
	}

	if date, err := time.ParseInLocation(layout, text, time.Local); err == nil {
		return date, nil
	}

	if date, err := time.ParseInLocation("02/01/2006", text, time.Local); err == nil {
		return date, nil
	}

	if date, err := time.ParseInLocation("02/01", text, time.Local); err == nil {
		return time.Date(today.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local), nil
	}

	return time.Time{}, fmt.Errorf("invalid date %q", text)
}

func clock(text string) (int, error) {
	raw := strings.ToLower(strings.TrimSpace(text))
	pm := strings.HasSuffix(raw, "pm")
	am := strings.HasSuffix(raw, "am")
	raw = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(raw, "pm"), "am"))

	parts := strings.Split(raw, ":")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid time %q", text)
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid time %q", text)
	}

	var minute int
	if len(parts) == 2 {
		minute, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("invalid time %q", text)
		}
	}

	if am || pm {
		if hour < 1 || hour > 12 {
			return 0, fmt.Errorf("invalid time %q", text)
		}
		hour %= 12
		if pm {
			hour += 12
		}
	}

	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, fmt.Errorf("invalid time %q", text)
	}

	return hour*60 + minute, nil
}

func span(text string) (int, int, error) {
	text = strings.TrimSpace(text)
	if text == "" || strings.EqualFold(text, "all") {
		return -1, -1, nil
	}

	left, right, found := strings.Cut(text, "-")

	begin, err := clock(left)
	if err != nil {
		return 0, 0, err
	}
	if !found {
		return begin, -1, nil
	}

	end, err := clock(right)
	if err != nil {
		return 0, 0, err
	}
	if end <= begin {
		return 0, 0, errors.New("end must be after start")
	}

	return begin, end, nil
}

func hhmm(minutes int) string {
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}

func slot(ev event) string {
	if ev.Begin < 0 {
		return ""
	}
	if ev.End < 0 {
		return hhmm(ev.Begin)
	}
	return hhmm(ev.Begin) + "-" + hhmm(ev.End)
}

func label(ev event) string {
	if text := slot(ev); text != "" {
		return text
	}
	return "all day"
}

func anchor(args []string) (time.Time, error) {
	if len(args) == 0 {
		return midnight(time.Now()), nil
	}
	return when(args[0], time.Now())
}

func open() (*bbolt.DB, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to locate config directory: %w", err)
	}

	dir = filepath.Join(dir, "mathrock")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := bbolt.Open(filepath.Join(dir, "calendar.db"), 0o600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucket)
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to prepare database: %w", err)
	}

	return db, nil
}

func update(fn func(*bbolt.Bucket) error) error {
	db, err := open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bbolt.Tx) error {
		return fn(tx.Bucket(bucket))
	})
}

func view(fn func(*bbolt.Bucket) error) error {
	db, err := open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.View(func(tx *bbolt.Tx) error {
		return fn(tx.Bucket(bucket))
	})
}

func load(b *bbolt.Bucket) ([]event, error) {
	var out []event

	err := b.ForEach(func(_, v []byte) error {
		var ev event
		if err := json.Unmarshal(v, &ev); err != nil {
			return fmt.Errorf("failed to decode event: %w", err)
		}
		out = append(out, ev)
		return nil
	})

	return out, err
}

func save(b *bbolt.Bucket, ev event) error {
	raw, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("failed to encode event: %w", err)
	}

	if err := b.Put(key(ev.ID), raw); err != nil {
		return fmt.Errorf("failed to save event: %w", err)
	}

	return nil
}

func order(in []event) []event {
	out := make([]event, len(in))
	copy(out, in)

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Day != out[j].Day {
			return out[i].Day < out[j].Day
		}
		if out[i].Begin != out[j].Begin {
			return out[i].Begin < out[j].Begin
		}
		return out[i].ID < out[j].ID
	})

	return out
}

func fetch() ([]event, error) {
	var out []event

	err := view(func(b *bbolt.Bucket) error {
		all, err := load(b)
		out = order(all)
		return err
	})

	return out, err
}

func numbers(args []string) choose {
	return func(all []event) ([]event, error) {
		var out []event
		seen := map[int]bool{}

		for _, arg := range args {
			num, err := strconv.Atoi(arg)
			if err != nil || num < 1 || num > len(all) {
				return nil, fmt.Errorf("no event numbered %q", arg)
			}
			if seen[num] {
				continue
			}
			seen[num] = true
			out = append(out, all[num-1])
		}

		return out, nil
	}
}

func ident(id uint64) choose {
	return func(all []event) ([]event, error) {
		for _, ev := range all {
			if ev.ID == id {
				return []event{ev}, nil
			}
		}
		return nil, errors.New("event no longer exists")
	}
}

func change(sel choose, fn func(*event)) ([]event, error) {
	var out []event

	err := update(func(b *bbolt.Bucket) error {
		all, err := load(b)
		if err != nil {
			return err
		}

		hit, err := sel(order(all))
		if err != nil {
			return err
		}

		for _, ev := range hit {
			fn(&ev)
			if err := save(b, ev); err != nil {
				return err
			}
			out = append(out, ev)
		}

		return nil
	})

	return out, err
}

func remove(sel choose) ([]event, error) {
	var hit []event

	err := update(func(b *bbolt.Bucket) error {
		all, err := load(b)
		if err != nil {
			return err
		}

		hit, err = sel(order(all))
		if err != nil {
			return err
		}

		for _, ev := range hit {
			if err := b.Delete(key(ev.ID)); err != nil {
				return fmt.Errorf("failed to delete event: %w", err)
			}
		}

		return nil
	})

	return hit, err
}

func insert(fresh event) (event, int, error) {
	var num int

	err := update(func(b *bbolt.Bucket) error {
		id, err := b.NextSequence()
		if err != nil {
			return fmt.Errorf("failed to allocate id: %w", err)
		}

		fresh.ID = id
		if err := save(b, fresh); err != nil {
			return err
		}

		all, err := load(b)
		if err != nil {
			return err
		}

		for i, cur := range order(all) {
			if cur.ID == id {
				num = i + 1
			}
		}

		return nil
	})

	return fresh, num, err
}

func Calendar() *cobra.Command {
	root := &cobra.Command{
		Use:          "calendar",
		Short:        "Keep track of events on a calendar",
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return dash()
		},
	}

	root.AddCommand(month(), week(), day(), ls(), add(), edit(), rm(), find(), prune(), tui())

	return root
}

func month() *cobra.Command {
	return &cobra.Command{
		Use:   "month [date]",
		Short: "Show a month with its events",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			at, err := anchor(args)
			if err != nil {
				return err
			}

			all, err := fetch()
			if err != nil {
				return err
			}

			prefix := at.Format("2006-01")
			lines := listing(all, func(ev event) bool { return strings.HasPrefix(ev.Day, prefix) })

			fmt.Println("\n" + grid(at, time.Time{}, marks(all), time.Now()) + "\n")
			if len(lines) > 0 {
				fmt.Println(strings.Join(lines, "\n") + "\n")
			}

			return nil
		},
	}
}

func week() *cobra.Command {
	return &cobra.Command{
		Use:   "week [date]",
		Short: "Show the week of a date, starting on Monday",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			at, err := anchor(args)
			if err != nil {
				return err
			}

			all, err := fetch()
			if err != nil {
				return err
			}

			start := at.AddDate(0, 0, -((int(at.Weekday()) + 6) % 7))

			var lines []string
			for i := 0; i < 7; i++ {
				lines = append(lines, block(start.AddDate(0, 0, i), all, true)...)
				lines = append(lines, "")
			}

			fmt.Println("\n" + strings.Join(lines, "\n"))
			return nil
		},
	}
}

func day() *cobra.Command {
	return &cobra.Command{
		Use:   "day [date]",
		Short: "Show a single day",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			at, err := anchor(args)
			if err != nil {
				return err
			}

			all, err := fetch()
			if err != nil {
				return err
			}

			fmt.Println("\n" + strings.Join(block(at, all, true), "\n") + "\n")
			return nil
		},
	}
}

func ls() *cobra.Command {
	return &cobra.Command{
		Use:   "ls [days]",
		Short: "List upcoming events, 14 days by default",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			ahead := 14
			if len(args) == 1 {
				num, err := strconv.Atoi(args[0])
				if err != nil || num < 1 {
					return fmt.Errorf("days must be a positive number, got %q", args[0])
				}
				ahead = num
			}

			all, err := fetch()
			if err != nil {
				return err
			}

			lines := upcoming(all, ahead)
			if len(lines) == 0 {
				fmt.Println(faint.Render(fmt.Sprintf("  no events in the next %d days", ahead)))
				return nil
			}

			fmt.Println("\n" + strings.Join(lines, "\n") + "\n")
			return nil
		},
	}
}

func add() *cobra.Command {
	var on, at, place string

	cmd := &cobra.Command{
		Use:   "add <title>",
		Short: "Add an event",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := strings.TrimSpace(strings.Join(args, " "))
			if name == "" {
				return errors.New("title is empty")
			}

			date, err := when(on, time.Now())
			if err != nil {
				return err
			}

			begin, end, err := span(at)
			if err != nil {
				return err
			}

			ev, num, err := insert(event{
				Title: name,
				Day:   date.Format(layout),
				Begin: begin,
				End:   end,
				Place: strings.TrimSpace(place),
			})
			if err != nil {
				return err
			}

			fmt.Println(faint.Render("  added ") + pink.Render(strconv.Itoa(num)) + " " + ev.Title + faint.Render("  "+date.Format("Mon, 02 Jan")+"  "+label(ev)))
			return nil
		},
	}

	cmd.Flags().StringVarP(&on, "on", "o", "today", "date: today, tomorrow, mon, +3, 2026-12-31 or 31/12")
	cmd.Flags().StringVarP(&at, "at", "a", "", "time: 14:00, 14:00-15:30 or 2pm, empty for all day")
	cmd.Flags().StringVarP(&place, "place", "p", "", "where it happens")

	return cmd
}

func edit() *cobra.Command {
	var on, at, place string

	cmd := &cobra.Command{
		Use:   "edit <number> [title]",
		Short: "Change an event",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			flags := c.Flags()
			name := strings.TrimSpace(strings.Join(args[1:], " "))

			if name == "" && !flags.Changed("on") && !flags.Changed("at") && !flags.Changed("place") {
				return errors.New("nothing to change")
			}

			var date time.Time
			var begin, end int
			var err error

			if flags.Changed("on") {
				date, err = when(on, time.Now())
				if err != nil {
					return err
				}
			}

			if flags.Changed("at") {
				begin, end, err = span(at)
				if err != nil {
					return err
				}
			}

			hit, err := change(numbers(args[:1]), func(ev *event) {
				if name != "" {
					ev.Title = name
				}
				if flags.Changed("on") {
					ev.Day = date.Format(layout)
				}
				if flags.Changed("at") {
					ev.Begin, ev.End = begin, end
				}
				if flags.Changed("place") {
					ev.Place = strings.TrimSpace(place)
				}
			})
			if err != nil {
				return err
			}

			return echo(hit)
		},
	}

	cmd.Flags().StringVarP(&on, "on", "o", "", "new date")
	cmd.Flags().StringVarP(&at, "at", "a", "", "new time, empty for all day")
	cmd.Flags().StringVarP(&place, "place", "p", "", "new place")

	return cmd
}

func rm() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <number>...",
		Short: "Remove events",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			hit, err := remove(numbers(args))
			if err != nil {
				return err
			}

			for _, ev := range hit {
				fmt.Println(faint.Render("  removed ") + ev.Title + faint.Render("  "+parse(ev.Day).Format("Mon, 02 Jan 2006")))
			}
			return nil
		},
	}
}

func find() *cobra.Command {
	return &cobra.Command{
		Use:   "find <query>",
		Short: "Search events by title or place",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			query := strings.ToLower(strings.Join(args, " "))

			all, err := fetch()
			if err != nil {
				return err
			}

			lines := listing(all, func(ev event) bool {
				return strings.Contains(strings.ToLower(ev.Title), query) || strings.Contains(strings.ToLower(ev.Place), query)
			})
			if len(lines) == 0 {
				fmt.Println(faint.Render("  no events match " + query))
				return nil
			}

			fmt.Println("\n" + strings.Join(lines, "\n") + "\n")
			return nil
		},
	}
}

func prune() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove every event before today",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if !yes {
				return errors.New("pruning requires the --yes flag")
			}

			var count int
			today := midnight(time.Now()).Format(layout)

			err := update(func(b *bbolt.Bucket) error {
				all, err := load(b)
				if err != nil {
					return err
				}

				for _, ev := range all {
					if ev.Day >= today {
						continue
					}
					if err := b.Delete(key(ev.ID)); err != nil {
						return fmt.Errorf("failed to delete event: %w", err)
					}
					count++
				}

				return nil
			})
			if err != nil {
				return err
			}

			fmt.Println(faint.Render("  pruned ") + strconv.Itoa(count) + faint.Render(" past events"))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "confirm the deletion")

	return cmd
}package calendar

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

const layout = "2006-01-02"

var bucket = []byte("calendar")

type event struct {
	ID    uint64 `json:"id"`
	Title string `json:"title"`
	Day   string `json:"day"`
	Begin int    `json:"begin"`
	End   int    `json:"end"`
	Place string `json:"place"`
}

type choose func([]event) ([]event, error)

func key(id uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, id)
	return buf
}

func midnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

func parse(stamp string) time.Time {
	out, err := time.ParseInLocation(layout, stamp, time.Local)
	if err != nil {
		return time.Time{}
	}
	return out
}

func shift(t time.Time, months int) time.Time {
	first := time.Date(t.Year(), t.Month()+time.Month(months), 1, 0, 0, 0, 0, time.Local)
	last := time.Date(first.Year(), first.Month()+1, 0, 0, 0, 0, 0, time.Local).Day()

	num := t.Day()
	if num > last {
		num = last
	}

	return time.Date(first.Year(), first.Month(), num, 0, 0, 0, 0, time.Local)
}

func when(text string, now time.Time) (time.Time, error) {
	text = strings.ToLower(strings.TrimSpace(text))
	today := midnight(now)

	switch text {
	case "", "today":
		return today, nil
	case "tomorrow":
		return today.AddDate(0, 0, 1), nil
	case "yesterday":
		return today.AddDate(0, 0, -1), nil
	}

	if text[0] == '+' || text[0] == '-' {
		if num, err := strconv.Atoi(text); err == nil {
			return today.AddDate(0, 0, num), nil
		}
	}

	for i := 0; i < 7; i++ {
		full := strings.ToLower(time.Weekday(i).String())
		if text == full || text == full[:3] {
			ahead := (i - int(today.Weekday()) + 7) % 7
			if ahead == 0 {
				ahead = 7
			}
			return today.AddDate(0, 0, ahead), nil
		}
	}

	if date, err := time.ParseInLocation(layout, text, time.Local); err == nil {
		return date, nil
	}

	if date, err := time.ParseInLocation("02/01/2006", text, time.Local); err == nil {
		return date, nil
	}

	if date, err := time.ParseInLocation("02/01", text, time.Local); err == nil {
		return time.Date(today.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local), nil
	}

	return time.Time{}, fmt.Errorf("invalid date %q", text)
}

func clock(text string) (int, error) {
	raw := strings.ToLower(strings.TrimSpace(text))
	pm := strings.HasSuffix(raw, "pm")
	am := strings.HasSuffix(raw, "am")
	raw = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(raw, "pm"), "am"))

	parts := strings.Split(raw, ":")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid time %q", text)
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid time %q", text)
	}

	var minute int
	if len(parts) == 2 {
		minute, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("invalid time %q", text)
		}
	}

	if am || pm {
		if hour < 1 || hour > 12 {
			return 0, fmt.Errorf("invalid time %q", text)
		}
		hour %= 12
		if pm {
			hour += 12
		}
	}

	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, fmt.Errorf("invalid time %q", text)
	}

	return hour*60 + minute, nil
}

func span(text string) (int, int, error) {
	text = strings.TrimSpace(text)
	if text == "" || strings.EqualFold(text, "all") {
		return -1, -1, nil
	}

	left, right, found := strings.Cut(text, "-")

	begin, err := clock(left)
	if err != nil {
		return 0, 0, err
	}
	if !found {
		return begin, -1, nil
	}

	end, err := clock(right)
	if err != nil {
		return 0, 0, err
	}
	if end <= begin {
		return 0, 0, errors.New("end must be after start")
	}

	return begin, end, nil
}

func hhmm(minutes int) string {
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}

func slot(ev event) string {
	if ev.Begin < 0 {
		return ""
	}
	if ev.End < 0 {
		return hhmm(ev.Begin)
	}
	return hhmm(ev.Begin) + "-" + hhmm(ev.End)
}

func label(ev event) string {
	if text := slot(ev); text != "" {
		return text
	}
	return "all day"
}

func anchor(args []string) (time.Time, error) {
	if len(args) == 0 {
		return midnight(time.Now()), nil
	}
	return when(args[0], time.Now())
}

func open() (*bbolt.DB, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to locate config directory: %w", err)
	}

	dir = filepath.Join(dir, "mathrock")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := bbolt.Open(filepath.Join(dir, "calendar.db"), 0o600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucket)
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to prepare database: %w", err)
	}

	return db, nil
}

func update(fn func(*bbolt.Bucket) error) error {
	db, err := open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bbolt.Tx) error {
		return fn(tx.Bucket(bucket))
	})
}

func view(fn func(*bbolt.Bucket) error) error {
	db, err := open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.View(func(tx *bbolt.Tx) error {
		return fn(tx.Bucket(bucket))
	})
}

func load(b *bbolt.Bucket) ([]event, error) {
	var out []event

	err := b.ForEach(func(_, v []byte) error {
		var ev event
		if err := json.Unmarshal(v, &ev); err != nil {
			return fmt.Errorf("failed to decode event: %w", err)
		}
		out = append(out, ev)
		return nil
	})

	return out, err
}

func save(b *bbolt.Bucket, ev event) error {
	raw, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("failed to encode event: %w", err)
	}

	if err := b.Put(key(ev.ID), raw); err != nil {
		return fmt.Errorf("failed to save event: %w", err)
	}

	return nil
}

func order(in []event) []event {
	out := make([]event, len(in))
	copy(out, in)

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Day != out[j].Day {
			return out[i].Day < out[j].Day
		}
		if out[i].Begin != out[j].Begin {
			return out[i].Begin < out[j].Begin
		}
		return out[i].ID < out[j].ID
	})

	return out
}

func fetch() ([]event, error) {
	var out []event

	err := view(func(b *bbolt.Bucket) error {
		all, err := load(b)
		out = order(all)
		return err
	})

	return out, err
}

func numbers(args []string) choose {
	return func(all []event) ([]event, error) {
		var out []event
		seen := map[int]bool{}

		for _, arg := range args {
			num, err := strconv.Atoi(arg)
			if err != nil || num < 1 || num > len(all) {
				return nil, fmt.Errorf("no event numbered %q", arg)
			}
			if seen[num] {
				continue
			}
			seen[num] = true
			out = append(out, all[num-1])
		}

		return out, nil
	}
}

func ident(id uint64) choose {
	return func(all []event) ([]event, error) {
		for _, ev := range all {
			if ev.ID == id {
				return []event{ev}, nil
			}
		}
		return nil, errors.New("event no longer exists")
	}
}

func change(sel choose, fn func(*event)) ([]event, error) {
	var out []event

	err := update(func(b *bbolt.Bucket) error {
		all, err := load(b)
		if err != nil {
			return err
		}

		hit, err := sel(order(all))
		if err != nil {
			return err
		}

		for _, ev := range hit {
			fn(&ev)
			if err := save(b, ev); err != nil {
				return err
			}
			out = append(out, ev)
		}

		return nil
	})

	return out, err
}

func remove(sel choose) ([]event, error) {
	var hit []event

	err := update(func(b *bbolt.Bucket) error {
		all, err := load(b)
		if err != nil {
			return err
		}

		hit, err = sel(order(all))
		if err != nil {
			return err
		}

		for _, ev := range hit {
			if err := b.Delete(key(ev.ID)); err != nil {
				return fmt.Errorf("failed to delete event: %w", err)
			}
		}

		return nil
	})

	return hit, err
}

func insert(fresh event) (event, int, error) {
	var num int

	err := update(func(b *bbolt.Bucket) error {
		id, err := b.NextSequence()
		if err != nil {
			return fmt.Errorf("failed to allocate id: %w", err)
		}

		fresh.ID = id
		if err := save(b, fresh); err != nil {
			return err
		}

		all, err := load(b)
		if err != nil {
			return err
		}

		for i, cur := range order(all) {
			if cur.ID == id {
				num = i + 1
			}
		}

		return nil
	})

	return fresh, num, err
}

func Calendar() *cobra.Command {
	root := &cobra.Command{
		Use:          "calendar",
		Short:        "Keep track of events on a calendar",
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return dash()
		},
	}

	root.AddCommand(month(), week(), day(), ls(), add(), edit(), rm(), find(), prune(), tui())

	return root
}

func month() *cobra.Command {
	return &cobra.Command{
		Use:   "month [date]",
		Short: "Show a month with its events",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			at, err := anchor(args)
			if err != nil {
				return err
			}

			all, err := fetch()
			if err != nil {
				return err
			}

			prefix := at.Format("2006-01")
			lines := listing(all, func(ev event) bool { return strings.HasPrefix(ev.Day, prefix) })

			fmt.Println("\n" + grid(at, time.Time{}, marks(all), time.Now()) + "\n")
			if len(lines) > 0 {
				fmt.Println(strings.Join(lines, "\n") + "\n")
			}

			return nil
		},
	}
}

func week() *cobra.Command {
	return &cobra.Command{
		Use:   "week [date]",
		Short: "Show the week of a date, starting on Monday",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			at, err := anchor(args)
			if err != nil {
				return err
			}

			all, err := fetch()
			if err != nil {
				return err
			}

			start := at.AddDate(0, 0, -((int(at.Weekday()) + 6) % 7))

			var lines []string
			for i := 0; i < 7; i++ {
				lines = append(lines, block(start.AddDate(0, 0, i), all, true)...)
				lines = append(lines, "")
			}

			fmt.Println("\n" + strings.Join(lines, "\n"))
			return nil
		},
	}
}

func day() *cobra.Command {
	return &cobra.Command{
		Use:   "day [date]",
		Short: "Show a single day",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			at, err := anchor(args)
			if err != nil {
				return err
			}

			all, err := fetch()
			if err != nil {
				return err
			}

			fmt.Println("\n" + strings.Join(block(at, all, true), "\n") + "\n")
			return nil
		},
	}
}

func ls() *cobra.Command {
	return &cobra.Command{
		Use:   "ls [days]",
		Short: "List upcoming events, 14 days by default",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			ahead := 14
			if len(args) == 1 {
				num, err := strconv.Atoi(args[0])
				if err != nil || num < 1 {
					return fmt.Errorf("days must be a positive number, got %q", args[0])
				}
				ahead = num
			}

			all, err := fetch()
			if err != nil {
				return err
			}

			lines := upcoming(all, ahead)
			if len(lines) == 0 {
				fmt.Println(faint.Render(fmt.Sprintf("  no events in the next %d days", ahead)))
				return nil
			}

			fmt.Println("\n" + strings.Join(lines, "\n") + "\n")
			return nil
		},
	}
}

func add() *cobra.Command {
	var on, at, place string

	cmd := &cobra.Command{
		Use:   "add <title>",
		Short: "Add an event",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := strings.TrimSpace(strings.Join(args, " "))
			if name == "" {
				return errors.New("title is empty")
			}

			date, err := when(on, time.Now())
			if err != nil {
				return err
			}

			begin, end, err := span(at)
			if err != nil {
				return err
			}

			ev, num, err := insert(event{
				Title: name,
				Day:   date.Format(layout),
				Begin: begin,
				End:   end,
				Place: strings.TrimSpace(place),
			})
			if err != nil {
				return err
			}

			fmt.Println(faint.Render("  added ") + pink.Render(strconv.Itoa(num)) + " " + ev.Title + faint.Render("  "+date.Format("Mon, 02 Jan")+"  "+label(ev)))
			return nil
		},
	}

	cmd.Flags().StringVarP(&on, "on", "o", "today", "date: today, tomorrow, mon, +3, 2026-12-31 or 31/12")
	cmd.Flags().StringVarP(&at, "at", "a", "", "time: 14:00, 14:00-15:30 or 2pm, empty for all day")
	cmd.Flags().StringVarP(&place, "place", "p", "", "where it happens")

	return cmd
}

func edit() *cobra.Command {
	var on, at, place string

	cmd := &cobra.Command{
		Use:   "edit <number> [title]",
		Short: "Change an event",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			flags := c.Flags()
			name := strings.TrimSpace(strings.Join(args[1:], " "))

			if name == "" && !flags.Changed("on") && !flags.Changed("at") && !flags.Changed("place") {
				return errors.New("nothing to change")
			}

			var date time.Time
			var begin, end int
			var err error

			if flags.Changed("on") {
				date, err = when(on, time.Now())
				if err != nil {
					return err
				}
			}

			if flags.Changed("at") {
				begin, end, err = span(at)
				if err != nil {
					return err
				}
			}

			hit, err := change(numbers(args[:1]), func(ev *event) {
				if name != "" {
					ev.Title = name
				}
				if flags.Changed("on") {
					ev.Day = date.Format(layout)
				}
				if flags.Changed("at") {
					ev.Begin, ev.End = begin, end
				}
				if flags.Changed("place") {
					ev.Place = strings.TrimSpace(place)
				}
			})
			if err != nil {
				return err
			}

			return echo(hit)
		},
	}

	cmd.Flags().StringVarP(&on, "on", "o", "", "new date")
	cmd.Flags().StringVarP(&at, "at", "a", "", "new time, empty for all day")
	cmd.Flags().StringVarP(&place, "place", "p", "", "new place")

	return cmd
}

func rm() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <number>...",
		Short: "Remove events",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			hit, err := remove(numbers(args))
			if err != nil {
				return err
			}

			for _, ev := range hit {
				fmt.Println(faint.Render("  removed ") + ev.Title + faint.Render("  "+parse(ev.Day).Format("Mon, 02 Jan 2006")))
			}
			return nil
		},
	}
}

func find() *cobra.Command {
	return &cobra.Command{
		Use:   "find <query>",
		Short: "Search events by title or place",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			query := strings.ToLower(strings.Join(args, " "))

			all, err := fetch()
			if err != nil {
				return err
			}

			lines := listing(all, func(ev event) bool {
				return strings.Contains(strings.ToLower(ev.Title), query) || strings.Contains(strings.ToLower(ev.Place), query)
			})
			if len(lines) == 0 {
				fmt.Println(faint.Render("  no events match " + query))
				return nil
			}

			fmt.Println("\n" + strings.Join(lines, "\n") + "\n")
			return nil
		},
	}
}

func prune() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove every event before today",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if !yes {
				return errors.New("pruning requires the --yes flag")
			}

			var count int
			today := midnight(time.Now()).Format(layout)

			err := update(func(b *bbolt.Bucket) error {
				all, err := load(b)
				if err != nil {
					return err
				}

				for _, ev := range all {
					if ev.Day >= today {
						continue
					}
					if err := b.Delete(key(ev.ID)); err != nil {
						return fmt.Errorf("failed to delete event: %w", err)
					}
					count++
				}

				return nil
			})
			if err != nil {
				return err
			}

			fmt.Println(faint.Render("  pruned ") + strconv.Itoa(count) + faint.Render(" past events"))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "confirm the deletion")

	return cmd
}
