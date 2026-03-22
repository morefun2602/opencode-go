package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var logoLeft = [4]string{
	"                   ",
	"█▀▀█ █▀▀█ █▀▀█ █▀▀▄",
	"█__█ █__█ █^^^ █__█",
	"▀▀▀▀ █▀▀▀ ▀▀▀▀ ▀~~▀",
}

var logoRight = [4]string{
	"             ▄     ",
	"█▀▀▀ █▀▀█ █▀▀█ █▀▀█",
	"█___ █__█ █__█ █^^^",
	"▀▀▀▀ ▀▀▀▀ ▀▀▀▀ ▀▀▀▀",
}

const logoMarks = "_^~"

func tintColor(base, overlay lipgloss.Color, alpha float64) lipgloss.Color {
	br, bg, bb := hexToRGB(string(base))
	or, og, ob := hexToRGB(string(overlay))
	r := int(float64(br) + (float64(or)-float64(br))*alpha)
	g := int(float64(bg) + (float64(og)-float64(bg))*alpha)
	b := int(float64(bb) + (float64(ob)-float64(bb))*alpha)
	return lipgloss.Color(rgbToHex(r, g, b))
}

func hexToRGB(hex string) (int, int, int) {
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return 200, 200, 200
	}
	r := hexVal(hex[0])<<4 | hexVal(hex[1])
	g := hexVal(hex[2])<<4 | hexVal(hex[3])
	b := hexVal(hex[4])<<4 | hexVal(hex[5])
	return r, g, b
}

func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return 0
}

func rgbToHex(r, g, b int) string {
	clamp := func(v int) int {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return v
	}
	const hex = "0123456789abcdef"
	r, g, b = clamp(r), clamp(g), clamp(b)
	return "#" + string([]byte{
		hex[r>>4], hex[r&0xf],
		hex[g>>4], hex[g&0xf],
		hex[b>>4], hex[b&0xf],
	})
}

// RenderLogo renders the OPENCODE ASCII art with shadow effects.
func RenderLogo(theme Theme) string {
	fg := theme.Subtle
	fgRight := theme.Text
	bg := theme.Background
	shadow := tintColor(bg, fg, 0.25)

	var rows []string
	for i := 0; i < 4; i++ {
		left := renderLogoLine(logoLeft[i], fg, shadow, bg)
		right := renderLogoLine(logoRight[i], fgRight, shadow, bg)
		rows = append(rows, left+"  "+right)
	}
	return strings.Join(rows, "\n")
}

func renderLogoLine(line string, fg, shadow, bg lipgloss.Color) string {
	var sb strings.Builder
	runes := []rune(line)
	for _, ch := range runes {
		if strings.ContainsRune(logoMarks, ch) {
			switch ch {
			case '_':
				sb.WriteString(lipgloss.NewStyle().Background(shadow).Render(" "))
			case '^':
				sb.WriteString(lipgloss.NewStyle().Foreground(fg).Background(shadow).Render("▀"))
			case '~':
				sb.WriteString(lipgloss.NewStyle().Foreground(shadow).Render("▀"))
			}
		} else if ch == ' ' {
			sb.WriteRune(' ')
		} else {
			sb.WriteString(lipgloss.NewStyle().Foreground(fg).Render(string(ch)))
		}
	}
	return sb.String()
}
