package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	skip "github.com/skip2/go-qrcode"
	"github.com/spf13/cobra"
	yeqown "github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

var (
	frame = lipgloss.NewStyle().
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	panel = lipgloss.NewStyle().
		Padding(1, 2).
		MarginLeft(2).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("238"))

	link = lipgloss.NewStyle().
		Foreground(lipgloss.Color("212")).
		Bold(true)

	note = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true).
		Width(35)

	muted = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	accent = lipgloss.NewStyle().
		Foreground(lipgloss.Color("212"))
)

func file(url, output, logo, color string) error {
	levels := []yeqown.EncodeOption{}
	styles := []standard.ImageOption{standard.WithFgColorRGBHex(color)}

	if logo != "" {
		levels = append(levels, yeqown.WithErrorCorrectionLevel(yeqown.ErrorCorrectionHighest))
		styles = append(styles, standard.WithLogoImageFilePNG(logo))
	}

	code, err := yeqown.New(url)
	if err != nil {
		return fmt.Errorf("failed to initialize qr code: %w", err)
	}

	writer, err := standard.New(output, styles...)
	if err != nil {
		return fmt.Errorf("failed to initialize image writer: %w", err)
	}

	if err := code.Save(writer); err != nil {
		return fmt.Errorf("failed to save qr code: %w", err)
	}

	return nil
}

func term(url, desc string) error {
	code, err := skip.New(url, skip.Medium)
	if err != nil {
		return fmt.Errorf("failed to build terminal qr code: %w", err)
	}

	rows := []string{link.Render(url)}
	if desc != "" {
		rows = append(rows, "", note.Render(desc))
	}

	left := frame.Render(code.ToSmallString(false))
	right := panel.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))

	fmt.Println("\n" + lipgloss.JoinHorizontal(lipgloss.Center, left, right) + "\n")

	return nil
}

func qr() *cobra.Command {
	var url, output, logo, color, desc string
	var show bool

	root := &cobra.Command{
		Use:          "qrgen",
		Short:        "Instantly generate QR codes from the terminal",
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			if output != "" {
				if err := file(url, output, logo, color); err != nil {
					return err
				}
			}

			if show {
				if err := term(url, desc); err != nil {
					return err
				}
			}

			if output != "" {
				fmt.Println(muted.Render("saved ") + accent.Render(output))
			}

			return nil
		},
	}

	root.Flags().StringVarP(&url, "url", "u", "", "target url to encode (required)")
	root.Flags().StringVarP(&output, "output", "o", "", "image path to save the code, for example output.png")
	root.Flags().StringVarP(&logo, "logo", "l", "", "png image to embed at the center of the code")
	root.Flags().StringVarP(&color, "color", "c", "#000000", "hex color of the saved code, for example #FF5733")
	root.Flags().StringVarP(&desc, "desc", "d", "", "description shown beside the code in the terminal")
	root.Flags().BoolVarP(&show, "show", "s", true, "render the code in the terminal")

	root.MarkFlagRequired("url")

	return root
}

func main() {
	if err := qr().Execute(); err != nil {
		os.Exit(1)
	}
}
