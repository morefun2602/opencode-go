package tui

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme holds the full color palette for the TUI, matching the TS version's field set.
type Theme struct {
	Name string `json:"-"`

	// Primary colors
	Primary   lipgloss.Color `json:"primary"`
	Secondary lipgloss.Color `json:"secondary"`
	Accent    lipgloss.Color `json:"accent"`

	// Status
	Error   lipgloss.Color `json:"error"`
	Warning lipgloss.Color `json:"warning"`
	Success lipgloss.Color `json:"success"`
	Info    lipgloss.Color `json:"info"`

	// Text
	Text     lipgloss.Color `json:"text"`
	Subtle   lipgloss.Color `json:"textMuted"`

	// Backgrounds
	Background      lipgloss.Color `json:"background"`
	BackgroundPanel lipgloss.Color `json:"backgroundPanel"`
	BackgroundElem  lipgloss.Color `json:"backgroundElement"`

	// Borders
	Border       lipgloss.Color `json:"border"`
	BorderActive lipgloss.Color `json:"borderActive"`
	BorderSubtle lipgloss.Color `json:"borderSubtle"`

	// Diff
	DiffAdded              lipgloss.Color `json:"diffAdded"`
	DiffRemoved            lipgloss.Color `json:"diffRemoved"`
	DiffContext            lipgloss.Color `json:"diffContext"`
	DiffHunkHeader         lipgloss.Color `json:"diffHunkHeader"`
	DiffHighlightAdded     lipgloss.Color `json:"diffHighlightAdded"`
	DiffHighlightRemoved   lipgloss.Color `json:"diffHighlightRemoved"`
	DiffAddedBg            lipgloss.Color `json:"diffAddedBg"`
	DiffRemovedBg          lipgloss.Color `json:"diffRemovedBg"`
	DiffContextBg          lipgloss.Color `json:"diffContextBg"`
	DiffLineNumber         lipgloss.Color `json:"diffLineNumber"`
	DiffAddedLineNumberBg  lipgloss.Color `json:"diffAddedLineNumberBg"`
	DiffRemovedLineNumberBg lipgloss.Color `json:"diffRemovedLineNumberBg"`

	// Syntax highlighting
	SyntaxComment     lipgloss.Color `json:"syntaxComment"`
	SyntaxKeyword     lipgloss.Color `json:"syntaxKeyword"`
	SyntaxFunction    lipgloss.Color `json:"syntaxFunction"`
	SyntaxVariable    lipgloss.Color `json:"syntaxVariable"`
	SyntaxString      lipgloss.Color `json:"syntaxString"`
	SyntaxNumber      lipgloss.Color `json:"syntaxNumber"`
	SyntaxType        lipgloss.Color `json:"syntaxType"`
	SyntaxOperator    lipgloss.Color `json:"syntaxOperator"`
	SyntaxPunctuation lipgloss.Color `json:"syntaxPunctuation"`

	// Legacy aliases used by existing code (mapped from new fields)
	ToolBorder    lipgloss.Color `json:"-"`
	ToolHeader    lipgloss.Color `json:"-"`
	DialogOverlay lipgloss.Color `json:"-"`
	HeaderBg      lipgloss.Color `json:"-"`
	FooterBg      lipgloss.Color `json:"-"`
}

// fillLegacy populates legacy fields from the expanded fields.
func (t *Theme) fillLegacy() {
	t.ToolBorder = t.BorderSubtle
	t.ToolHeader = t.BackgroundElem
	t.DialogOverlay = t.BackgroundPanel
	t.HeaderBg = t.BackgroundElem
	t.FooterBg = t.BackgroundElem
	if t.ToolBorder == "" {
		t.ToolBorder = t.Border
	}
	if t.ToolHeader == "" {
		t.ToolHeader = t.Background
	}
	if t.DialogOverlay == "" {
		t.DialogOverlay = t.Background
	}
	if t.HeaderBg == "" {
		t.HeaderBg = t.Background
	}
	if t.FooterBg == "" {
		t.FooterBg = t.Background
	}
}

// IsDark returns true if this is a dark theme (heuristic based on background luminance).
func (t *Theme) IsDark() bool {
	r, g, b := hexToRGB(string(t.Background))
	lum := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
	return lum < 128
}

// --- Built-in themes ---

var Dark = func() Theme {
	t := Theme{
		Name:            "dark",
		Primary:         "#7C3AED",
		Secondary:       "#3B82F6",
		Accent:          "#F5C2E7",
		Error:           "#F38BA8",
		Warning:         "#FAB387",
		Success:         "#A6E3A1",
		Info:            "#94E2D5",
		Text:            "#CDD6F4",
		Subtle:          "#6C7086",
		Background:      "#1E1E2E",
		BackgroundPanel: "#181825",
		BackgroundElem:  "#313244",
		Border:          "#45475A",
		BorderActive:    "#585B70",
		BorderSubtle:    "#313244",
		DiffAdded:       "#A6E3A1", DiffRemoved: "#F38BA8",
		DiffContext: "#9399B2", DiffHunkHeader: "#FAB387",
		DiffHighlightAdded: "#A6E3A1", DiffHighlightRemoved: "#F38BA8",
		DiffAddedBg: "#24312B", DiffRemovedBg: "#3C2A32",
		DiffContextBg: "#181825", DiffLineNumber: "#45475A",
		DiffAddedLineNumberBg: "#1E2A25", DiffRemovedLineNumberBg: "#32232A",
		SyntaxComment: "#9399B2", SyntaxKeyword: "#CBA6F7",
		SyntaxFunction: "#89B4FA", SyntaxVariable: "#F38BA8",
		SyntaxString: "#A6E3A1", SyntaxNumber: "#FAB387",
		SyntaxType: "#F9E2AF", SyntaxOperator: "#89DCEB",
		SyntaxPunctuation: "#CDD6F4",
	}
	t.fillLegacy()
	return t
}()

var Light = func() Theme {
	t := Theme{
		Name:            "light",
		Primary:         "#7C3AED",
		Secondary:       "#3B82F6",
		Accent:          "#EA76CB",
		Error:           "#D20F39",
		Warning:         "#FE640B",
		Success:         "#40A02B",
		Info:            "#179299",
		Text:            "#4C4F69",
		Subtle:          "#9CA0B0",
		Background:      "#EFF1F5",
		BackgroundPanel: "#E6E9EF",
		BackgroundElem:  "#DCE0E8",
		Border:          "#BCC0CC",
		BorderActive:    "#ACB0BE",
		BorderSubtle:    "#CCD0DA",
		DiffAdded:       "#40A02B", DiffRemoved: "#D20F39",
		DiffContext: "#7C7F93", DiffHunkHeader: "#FE640B",
		DiffHighlightAdded: "#40A02B", DiffHighlightRemoved: "#D20F39",
		DiffAddedBg: "#D6F0D9", DiffRemovedBg: "#F6DFE2",
		DiffContextBg: "#E6E9EF", DiffLineNumber: "#BCC0CC",
		DiffAddedLineNumberBg: "#C9E3CB", DiffRemovedLineNumberBg: "#E9D3D6",
		SyntaxComment: "#7C7F93", SyntaxKeyword: "#8839EF",
		SyntaxFunction: "#1E66F5", SyntaxVariable: "#D20F39",
		SyntaxString: "#40A02B", SyntaxNumber: "#FE640B",
		SyntaxType: "#DF8E1D", SyntaxOperator: "#04A5E5",
		SyntaxPunctuation: "#4C4F69",
	}
	t.fillLegacy()
	return t
}()

var Catppuccin = func() Theme {
	t := Theme{
		Name: "catppuccin", Primary: "#89B4FA", Secondary: "#CBA6F7", Accent: "#F5C2E7",
		Error: "#F38BA8", Warning: "#F9E2AF", Success: "#A6E3A1", Info: "#94E2D5",
		Text: "#CDD6F4", Subtle: "#BAC2DE",
		Background: "#1E1E2E", BackgroundPanel: "#181825", BackgroundElem: "#11111B",
		Border: "#313244", BorderActive: "#45475A", BorderSubtle: "#585B70",
		DiffAdded: "#A6E3A1", DiffRemoved: "#F38BA8",
		DiffContext: "#9399B2", DiffHunkHeader: "#FAB387",
		DiffHighlightAdded: "#A6E3A1", DiffHighlightRemoved: "#F38BA8",
		DiffAddedBg: "#24312B", DiffRemovedBg: "#3C2A32",
		DiffContextBg: "#181825", DiffLineNumber: "#45475A",
		DiffAddedLineNumberBg: "#1E2A25", DiffRemovedLineNumberBg: "#32232A",
		SyntaxComment: "#9399B2", SyntaxKeyword: "#CBA6F7",
		SyntaxFunction: "#89B4FA", SyntaxVariable: "#F38BA8",
		SyntaxString: "#A6E3A1", SyntaxNumber: "#FAB387",
		SyntaxType: "#F9E2AF", SyntaxOperator: "#89DCEB",
		SyntaxPunctuation: "#CDD6F4",
	}
	t.fillLegacy()
	return t
}()

var Dracula = func() Theme {
	t := Theme{
		Name: "dracula", Primary: "#BD93F9", Secondary: "#FF79C6", Accent: "#8BE9FD",
		Error: "#FF5555", Warning: "#F1FA8C", Success: "#50FA7B", Info: "#FFB86C",
		Text: "#F8F8F2", Subtle: "#6272A4",
		Background: "#282A36", BackgroundPanel: "#21222C", BackgroundElem: "#44475A",
		Border: "#44475A", BorderActive: "#BD93F9", BorderSubtle: "#191A21",
		DiffAdded: "#50FA7B", DiffRemoved: "#FF5555",
		DiffContext: "#6272A4", DiffHunkHeader: "#6272A4",
		DiffHighlightAdded: "#50FA7B", DiffHighlightRemoved: "#FF5555",
		DiffAddedBg: "#1A3A1A", DiffRemovedBg: "#3A1A1A",
		DiffContextBg: "#21222C", DiffLineNumber: "#44475A",
		DiffAddedLineNumberBg: "#1A3A1A", DiffRemovedLineNumberBg: "#3A1A1A",
		SyntaxComment: "#6272A4", SyntaxKeyword: "#FF79C6",
		SyntaxFunction: "#50FA7B", SyntaxVariable: "#F8F8F2",
		SyntaxString: "#F1FA8C", SyntaxNumber: "#BD93F9",
		SyntaxType: "#8BE9FD", SyntaxOperator: "#FF79C6",
		SyntaxPunctuation: "#F8F8F2",
	}
	t.fillLegacy()
	return t
}()

var Nord = func() Theme {
	t := Theme{
		Name: "nord", Primary: "#88C0D0", Secondary: "#81A1C1", Accent: "#8FBCBB",
		Error: "#BF616A", Warning: "#D08770", Success: "#A3BE8C", Info: "#88C0D0",
		Text: "#ECEFF4", Subtle: "#8B95A7",
		Background: "#2E3440", BackgroundPanel: "#3B4252", BackgroundElem: "#434C5E",
		Border: "#434C5E", BorderActive: "#4C566A", BorderSubtle: "#434C5E",
		DiffAdded: "#A3BE8C", DiffRemoved: "#BF616A",
		DiffContext: "#8B95A7", DiffHunkHeader: "#8B95A7",
		DiffHighlightAdded: "#A3BE8C", DiffHighlightRemoved: "#BF616A",
		DiffAddedBg: "#3B4252", DiffRemovedBg: "#3B4252",
		DiffContextBg: "#3B4252", DiffLineNumber: "#434C5E",
		DiffAddedLineNumberBg: "#3B4252", DiffRemovedLineNumberBg: "#3B4252",
		SyntaxComment: "#8B95A7", SyntaxKeyword: "#81A1C1",
		SyntaxFunction: "#88C0D0", SyntaxVariable: "#8FBCBB",
		SyntaxString: "#A3BE8C", SyntaxNumber: "#B48EAD",
		SyntaxType: "#8FBCBB", SyntaxOperator: "#81A1C1",
		SyntaxPunctuation: "#D8DEE9",
	}
	t.fillLegacy()
	return t
}()

var TokyoNight = func() Theme {
	t := Theme{
		Name: "tokyonight", Primary: "#82AAFF", Secondary: "#C099FF", Accent: "#FF966C",
		Error: "#FF757F", Warning: "#FF966C", Success: "#C3E88D", Info: "#82AAFF",
		Text: "#C8D3F5", Subtle: "#828BB8",
		Background: "#1A1B26", BackgroundPanel: "#1E2030", BackgroundElem: "#222436",
		Border: "#737AA2", BorderActive: "#9099B2", BorderSubtle: "#545C7E",
		DiffAdded: "#4FD6BE", DiffRemoved: "#C53B53",
		DiffContext: "#828BB8", DiffHunkHeader: "#828BB8",
		DiffHighlightAdded: "#B8DB87", DiffHighlightRemoved: "#E26A75",
		DiffAddedBg: "#20303B", DiffRemovedBg: "#37222C",
		DiffContextBg: "#1E2030", DiffLineNumber: "#222436",
		DiffAddedLineNumberBg: "#1B2B34", DiffRemovedLineNumberBg: "#2D1F26",
		SyntaxComment: "#828BB8", SyntaxKeyword: "#C099FF",
		SyntaxFunction: "#82AAFF", SyntaxVariable: "#FF757F",
		SyntaxString: "#C3E88D", SyntaxNumber: "#FF966C",
		SyntaxType: "#FFC777", SyntaxOperator: "#86E1FC",
		SyntaxPunctuation: "#C8D3F5",
	}
	t.fillLegacy()
	return t
}()

var Gruvbox = func() Theme {
	t := Theme{
		Name: "gruvbox", Primary: "#FE8019", Secondary: "#D3869B", Accent: "#83A598",
		Error: "#FB4934", Warning: "#FABD2F", Success: "#B8BB26", Info: "#8EC07C",
		Text: "#EBDBB2", Subtle: "#928374",
		Background: "#282828", BackgroundPanel: "#1D2021", BackgroundElem: "#3C3836",
		Border: "#504945", BorderActive: "#665C54", BorderSubtle: "#3C3836",
		DiffAdded: "#B8BB26", DiffRemoved: "#FB4934",
		DiffContext: "#928374", DiffHunkHeader: "#928374",
		DiffHighlightAdded: "#B8BB26", DiffHighlightRemoved: "#FB4934",
		DiffAddedBg: "#2E3B2E", DiffRemovedBg: "#3B2E2E",
		DiffContextBg: "#1D2021", DiffLineNumber: "#504945",
		DiffAddedLineNumberBg: "#2E3B2E", DiffRemovedLineNumberBg: "#3B2E2E",
		SyntaxComment: "#928374", SyntaxKeyword: "#FB4934",
		SyntaxFunction: "#B8BB26", SyntaxVariable: "#83A598",
		SyntaxString: "#B8BB26", SyntaxNumber: "#D3869B",
		SyntaxType: "#FABD2F", SyntaxOperator: "#FE8019",
		SyntaxPunctuation: "#EBDBB2",
	}
	t.fillLegacy()
	return t
}()

var Monokai = func() Theme {
	t := Theme{
		Name: "monokai", Primary: "#F92672", Secondary: "#AE81FF", Accent: "#66D9EF",
		Error: "#F92672", Warning: "#FD971F", Success: "#A6E22E", Info: "#66D9EF",
		Text: "#F8F8F2", Subtle: "#75715E",
		Background: "#272822", BackgroundPanel: "#1E1F1C", BackgroundElem: "#3E3D32",
		Border: "#49483E", BorderActive: "#75715E", BorderSubtle: "#3E3D32",
		DiffAdded: "#A6E22E", DiffRemoved: "#F92672",
		DiffContext: "#75715E", DiffHunkHeader: "#75715E",
		DiffHighlightAdded: "#A6E22E", DiffHighlightRemoved: "#F92672",
		DiffAddedBg: "#2E3B2E", DiffRemovedBg: "#3B2E2E",
		DiffContextBg: "#1E1F1C", DiffLineNumber: "#49483E",
		DiffAddedLineNumberBg: "#2E3B2E", DiffRemovedLineNumberBg: "#3B2E2E",
		SyntaxComment: "#75715E", SyntaxKeyword: "#F92672",
		SyntaxFunction: "#A6E22E", SyntaxVariable: "#F8F8F2",
		SyntaxString: "#E6DB74", SyntaxNumber: "#AE81FF",
		SyntaxType: "#66D9EF", SyntaxOperator: "#F92672",
		SyntaxPunctuation: "#F8F8F2",
	}
	t.fillLegacy()
	return t
}()

var Solarized = func() Theme {
	t := Theme{
		Name: "solarized", Primary: "#268BD2", Secondary: "#6C71C4", Accent: "#2AA198",
		Error: "#DC322F", Warning: "#CB4B16", Success: "#859900", Info: "#2AA198",
		Text: "#839496", Subtle: "#586E75",
		Background: "#002B36", BackgroundPanel: "#073642", BackgroundElem: "#073642",
		Border: "#586E75", BorderActive: "#657B83", BorderSubtle: "#073642",
		DiffAdded: "#859900", DiffRemoved: "#DC322F",
		DiffContext: "#586E75", DiffHunkHeader: "#586E75",
		DiffHighlightAdded: "#859900", DiffHighlightRemoved: "#DC322F",
		DiffAddedBg: "#073642", DiffRemovedBg: "#073642",
		DiffContextBg: "#073642", DiffLineNumber: "#073642",
		DiffAddedLineNumberBg: "#073642", DiffRemovedLineNumberBg: "#073642",
		SyntaxComment: "#586E75", SyntaxKeyword: "#859900",
		SyntaxFunction: "#268BD2", SyntaxVariable: "#B58900",
		SyntaxString: "#2AA198", SyntaxNumber: "#D33682",
		SyntaxType: "#B58900", SyntaxOperator: "#859900",
		SyntaxPunctuation: "#839496",
	}
	t.fillLegacy()
	return t
}()

var RosePine = func() Theme {
	t := Theme{
		Name: "rosepine", Primary: "#C4A7E7", Secondary: "#F6C177", Accent: "#EBBCBA",
		Error: "#EB6F92", Warning: "#F6C177", Success: "#9CCFD8", Info: "#31748F",
		Text: "#E0DEF4", Subtle: "#908CAA",
		Background: "#191724", BackgroundPanel: "#1F1D2E", BackgroundElem: "#26233A",
		Border: "#403D52", BorderActive: "#524F67", BorderSubtle: "#26233A",
		DiffAdded: "#9CCFD8", DiffRemoved: "#EB6F92",
		DiffContext: "#908CAA", DiffHunkHeader: "#908CAA",
		DiffHighlightAdded: "#9CCFD8", DiffHighlightRemoved: "#EB6F92",
		DiffAddedBg: "#1F2D30", DiffRemovedBg: "#2D1F27",
		DiffContextBg: "#1F1D2E", DiffLineNumber: "#26233A",
		DiffAddedLineNumberBg: "#1F2D30", DiffRemovedLineNumberBg: "#2D1F27",
		SyntaxComment: "#908CAA", SyntaxKeyword: "#C4A7E7",
		SyntaxFunction: "#EBBCBA", SyntaxVariable: "#E0DEF4",
		SyntaxString: "#F6C177", SyntaxNumber: "#EB6F92",
		SyntaxType: "#9CCFD8", SyntaxOperator: "#C4A7E7",
		SyntaxPunctuation: "#E0DEF4",
	}
	t.fillLegacy()
	return t
}()

var OneDark = func() Theme {
	t := Theme{
		Name: "one-dark", Primary: "#61AFEF", Secondary: "#C678DD", Accent: "#56B6C2",
		Error: "#E06C75", Warning: "#D19A66", Success: "#98C379", Info: "#56B6C2",
		Text: "#ABB2BF", Subtle: "#5C6370",
		Background: "#282C34", BackgroundPanel: "#21252B", BackgroundElem: "#2C313A",
		Border: "#3E4452", BorderActive: "#4B5263", BorderSubtle: "#2C313A",
		DiffAdded: "#98C379", DiffRemoved: "#E06C75",
		DiffContext: "#5C6370", DiffHunkHeader: "#5C6370",
		DiffHighlightAdded: "#98C379", DiffHighlightRemoved: "#E06C75",
		DiffAddedBg: "#2A3D2A", DiffRemovedBg: "#3D2A2A",
		DiffContextBg: "#21252B", DiffLineNumber: "#3E4452",
		DiffAddedLineNumberBg: "#2A3D2A", DiffRemovedLineNumberBg: "#3D2A2A",
		SyntaxComment: "#5C6370", SyntaxKeyword: "#C678DD",
		SyntaxFunction: "#61AFEF", SyntaxVariable: "#E06C75",
		SyntaxString: "#98C379", SyntaxNumber: "#D19A66",
		SyntaxType: "#E5C07B", SyntaxOperator: "#56B6C2",
		SyntaxPunctuation: "#ABB2BF",
	}
	t.fillLegacy()
	return t
}()

var GitHub = func() Theme {
	t := Theme{
		Name: "github", Primary: "#0969DA", Secondary: "#8250DF", Accent: "#1F883D",
		Error: "#CF222E", Warning: "#BF8700", Success: "#1A7F37", Info: "#0550AE",
		Text: "#1F2328", Subtle: "#656D76",
		Background: "#FFFFFF", BackgroundPanel: "#F6F8FA", BackgroundElem: "#EAEEF2",
		Border: "#D0D7DE", BorderActive: "#0969DA", BorderSubtle: "#EAEEF2",
		DiffAdded: "#1A7F37", DiffRemoved: "#CF222E",
		DiffContext: "#656D76", DiffHunkHeader: "#656D76",
		DiffHighlightAdded: "#1A7F37", DiffHighlightRemoved: "#CF222E",
		DiffAddedBg: "#DAFBE1", DiffRemovedBg: "#FFD7D5",
		DiffContextBg: "#F6F8FA", DiffLineNumber: "#D0D7DE",
		DiffAddedLineNumberBg: "#CCFFD8", DiffRemovedLineNumberBg: "#FFC1BA",
		SyntaxComment: "#6E7781", SyntaxKeyword: "#CF222E",
		SyntaxFunction: "#8250DF", SyntaxVariable: "#953800",
		SyntaxString: "#0A3069", SyntaxNumber: "#0550AE",
		SyntaxType: "#0550AE", SyntaxOperator: "#CF222E",
		SyntaxPunctuation: "#1F2328",
	}
	t.fillLegacy()
	return t
}()

// BuiltinThemes maps theme names to their Theme values.
var BuiltinThemes = map[string]Theme{
	"dark":       Dark,
	"light":      Light,
	"catppuccin": Catppuccin,
	"dracula":    Dracula,
	"nord":       Nord,
	"tokyonight": TokyoNight,
	"gruvbox":    Gruvbox,
	"monokai":    Monokai,
	"solarized":  Solarized,
	"rosepine":   RosePine,
	"one-dark":   OneDark,
	"github":     GitHub,
}

// ThemeNames returns sorted theme name list.
func ThemeNames() []string {
	return []string{
		"dark", "light", "catppuccin", "dracula", "nord", "tokyonight",
		"gruvbox", "monokai", "solarized", "rosepine", "one-dark", "github",
	}
}

// ResolveTheme loads a theme by name (built-in) or from a JSON file path.
func ResolveTheme(nameOrPath string) Theme {
	if nameOrPath == "" {
		return Dark
	}
	lower := strings.ToLower(nameOrPath)
	if t, ok := BuiltinThemes[lower]; ok {
		return t
	}
	if t, err := LoadThemeFromFile(nameOrPath); err == nil {
		return t
	}
	return Dark
}

// themeJSON is the on-disk JSON format (matching the TS schema simplified for dark-only).
type themeJSON struct {
	Defs  map[string]string    `json:"defs"`
	Theme map[string]jsonColor `json:"theme"`
}

// jsonColor can be a plain string or {dark:"...", light:"..."}.
type jsonColor struct {
	value string
	dark  string
	light string
}

func (c *jsonColor) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		c.value = s
		return nil
	}
	var obj struct {
		Dark  string `json:"dark"`
		Light string `json:"light"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	c.dark = obj.Dark
	c.light = obj.Light
	return nil
}

func (c jsonColor) resolve(defs map[string]string, preferDark bool) string {
	raw := c.value
	if raw == "" {
		if preferDark {
			raw = c.dark
		} else {
			raw = c.light
		}
	}
	if raw == "" {
		raw = c.dark
	}
	if v, ok := defs[raw]; ok {
		return v
	}
	return raw
}

// LoadThemeFromFile reads a theme JSON file.
func LoadThemeFromFile(path string) (Theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Dark, err
	}
	return ParseThemeJSON(data, true)
}

// ParseThemeJSON parses theme JSON bytes into a Theme.
func ParseThemeJSON(data []byte, preferDark bool) (Theme, error) {
	var tj themeJSON
	if err := json.Unmarshal(data, &tj); err != nil {
		return Dark, err
	}

	get := func(key string) lipgloss.Color {
		if c, ok := tj.Theme[key]; ok {
			return lipgloss.Color(c.resolve(tj.Defs, preferDark))
		}
		return ""
	}

	t := Theme{
		Name:            "custom",
		Primary:         get("primary"),
		Secondary:       get("secondary"),
		Accent:          get("accent"),
		Error:           get("error"),
		Warning:         get("warning"),
		Success:         get("success"),
		Info:            get("info"),
		Text:            get("text"),
		Subtle:          get("textMuted"),
		Background:      get("background"),
		BackgroundPanel: get("backgroundPanel"),
		BackgroundElem:  get("backgroundElement"),
		Border:          get("border"),
		BorderActive:    get("borderActive"),
		BorderSubtle:    get("borderSubtle"),
		DiffAdded:       get("diffAdded"), DiffRemoved: get("diffRemoved"),
		DiffContext: get("diffContext"), DiffHunkHeader: get("diffHunkHeader"),
		DiffHighlightAdded: get("diffHighlightAdded"), DiffHighlightRemoved: get("diffHighlightRemoved"),
		DiffAddedBg: get("diffAddedBg"), DiffRemovedBg: get("diffRemovedBg"),
		DiffContextBg: get("diffContextBg"), DiffLineNumber: get("diffLineNumber"),
		DiffAddedLineNumberBg: get("diffAddedLineNumberBg"), DiffRemovedLineNumberBg: get("diffRemovedLineNumberBg"),
		SyntaxComment: get("syntaxComment"), SyntaxKeyword: get("syntaxKeyword"),
		SyntaxFunction: get("syntaxFunction"), SyntaxVariable: get("syntaxVariable"),
		SyntaxString: get("syntaxString"), SyntaxNumber: get("syntaxNumber"),
		SyntaxType: get("syntaxType"), SyntaxOperator: get("syntaxOperator"),
		SyntaxPunctuation: get("syntaxPunctuation"),
	}
	t.fillLegacy()
	return t, nil
}
