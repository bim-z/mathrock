package friendzone

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/Masterminds/squirrel"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var (
	sq = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question)

	client = &http.Client{Timeout: 10 * time.Second}

	frame = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00ffcc")).
		Padding(1, 3)

	accent = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00ffcc"))

	place = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ccff"))

	clock = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ff00ff"))

	muted = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	head = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ffffff")).
		Padding(0, 1)

	cell = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cccccc")).
		Padding(0, 1)

	stamp = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ff00ff")).
		Padding(0, 1)

	line = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffcc"))
)

type geo struct {
	Results []struct {
		Name     string  `json:"name"`
		Country  string  `json:"country"`
		Lat      float64 `json:"latitude"`
		Lon      float64 `json:"longitude"`
		Timezone string  `json:"timezone"`
	} `json:"results"`
}

func open() (*sql.DB, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to locate config directory: %w", err)
	}

	dir = filepath.Join(dir, "mathrock")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := sql.Open("sqlite3", filepath.Join(dir, "friendzone.db"))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS friends (
		name TEXT PRIMARY KEY,
		city TEXT NOT NULL,
		country TEXT NOT NULL,
		lat REAL NOT NULL,
		lon REAL NOT NULL,
		timezone TEXT NOT NULL
	)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to prepare database: %w", err)
	}

	return db, nil
}

func run(fn func(*sql.DB, []string) error) func(*cobra.Command, []string) error {
	return func(_ *cobra.Command, args []string) error {
		db, err := open()
		if err != nil {
			return err
		}
		defer db.Close()

		return fn(db, args)
	}
}

func Friend() *cobra.Command {
	root := &cobra.Command{
		Use:          "friendzone",
		Short:        "Track the local time of your friends around the world",
		SilenceUsage: true,
	}

	root.AddCommand(add(), rm(), show(), ls())

	return root
}

func add() *cobra.Command {
	return &cobra.Command{
		Use:   "add <name> <city>",
		Short: "Add a friend and their city",
		Args:  cobra.ExactArgs(2),
		RunE: run(func(db *sql.DB, args []string) error {
			name, city := args[0], args[1]

			resp, err := client.Get("https://geocoding-api.open-meteo.com/v1/search?count=1&name=" + url.QueryEscape(city))
			if err != nil {
				return fmt.Errorf("failed to reach geocoding api: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("geocoding api returned status %d", resp.StatusCode)
			}

			var out geo
			if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
				return fmt.Errorf("failed to decode api response: %w", err)
			}

			if len(out.Results) == 0 {
				return fmt.Errorf("city %q not found", city)
			}

			loc := out.Results[0]
			if _, err := time.LoadLocation(loc.Timezone); err != nil || loc.Timezone == "" {
				return fmt.Errorf("invalid timezone %q for city %q", loc.Timezone, city)
			}

			_, err = sq.Insert("friends").
				Columns("name", "city", "country", "lat", "lon", "timezone").
				Values(name, loc.Name, loc.Country, loc.Lat, loc.Lon, loc.Timezone).
				RunWith(db).
				Exec()
			if err != nil {
				if strings.Contains(err.Error(), "UNIQUE") {
					return fmt.Errorf("friend %q already exists", name)
				}
				return fmt.Errorf("failed to save friend: %w", err)
			}

			fmt.Println(muted.Render("added ") + accent.Render(name) + muted.Render("  "+loc.Name+", "+loc.Country+"  "+loc.Timezone))
			return nil
		}),
	}
}

func rm() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <name>",
		Short: "Remove a friend",
		Args:  cobra.ExactArgs(1),
		RunE: run(func(db *sql.DB, args []string) error {
			name := args[0]

			res, err := sq.Delete("friends").
				Where(squirrel.Eq{"name": name}).
				RunWith(db).
				Exec()
			if err != nil {
				return fmt.Errorf("failed to delete friend: %w", err)
			}

			affected, err := res.RowsAffected()
			if err != nil {
				return fmt.Errorf("failed to check affected rows: %w", err)
			}

			if affected == 0 {
				return fmt.Errorf("friend %q not found", name)
			}

			fmt.Println(muted.Render("removed ") + accent.Render(name))
			return nil
		}),
	}
}

func show() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show the current time of a friend",
		Args:  cobra.ExactArgs(1),
		RunE: run(func(db *sql.DB, args []string) error {
			name := args[0]

			var city, country, zone string
			err := sq.Select("city", "country", "timezone").
				From("friends").
				Where(squirrel.Eq{"name": name}).
				RunWith(db).
				QueryRow().
				Scan(&city, &country, &zone)
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("friend %q not found", name)
			}
			if err != nil {
				return fmt.Errorf("failed to query database: %w", err)
			}

			location, err := time.LoadLocation(zone)
			if err != nil {
				return fmt.Errorf("failed to load timezone %q: %w", zone, err)
			}

			now := time.Now().In(location)

			body := lipgloss.JoinVertical(lipgloss.Left,
				accent.Render(name),
				place.Render(city+", "+country),
				"",
				clock.Render(now.Format("15:04:05"))+muted.Render("  "+now.Format("Mon, 02 Jan 2006")),
			)

			fmt.Println("\n" + frame.Render(body) + "\n")
			return nil
		}),
	}
}

func ls() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List all friends and their local times",
		Args:  cobra.NoArgs,
		RunE: run(func(db *sql.DB, _ []string) error {
			res, err := sq.Select("name", "city", "country", "lat", "lon", "timezone").
				From("friends").
				OrderBy("name").
				RunWith(db).
				Query()
			if err != nil {
				return fmt.Errorf("failed to fetch friends: %w", err)
			}
			defer res.Close()

			var data [][]string
			now := time.Now()

			for res.Next() {
				var name, city, country, zone string
				var lat, lon float64

				if err := res.Scan(&name, &city, &country, &lat, &lon, &zone); err != nil {
					return fmt.Errorf("failed to read row: %w", err)
				}

				text := "-"
				if location, err := time.LoadLocation(zone); err == nil {
					text = now.In(location).Format("15:04:05")
				}

				data = append(data, []string{
					name,
					city + ", " + country,
					zone,
					text,
					fmt.Sprintf("%.2f, %.2f", lat, lon),
				})
			}

			if err := res.Err(); err != nil {
				return fmt.Errorf("failed to iterate rows: %w", err)
			}

			if len(data) == 0 {
				fmt.Println(muted.Render("no friends yet, add one with 'friendzone add'"))
				return nil
			}

			grid := table.New().
				Border(lipgloss.RoundedBorder()).
				BorderStyle(line).
				Headers("Name", "Location", "Timezone", "Time", "Coordinates").
				Rows(data...).
				StyleFunc(func(row, col int) lipgloss.Style {
					switch {
					case row == table.HeaderRow:
						return head
					case col == 3:
						return stamp
					default:
						return cell
					}
				})

			fmt.Println(grid.Render())
			return nil
		}),
	}
}
