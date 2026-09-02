package config

// Palette holds the TUI's color values, each a lipgloss-compatible color
// spec: an ANSI 256 color number as a string (e.g. "212"), or a hex code
// (e.g. "#ff69b4"). Defaults match the values gh-custom-properties has
// always used; set any of the color_* keys in the config file to override.
type Palette struct {
	Title    string
	Header   string
	Cursor   string
	Selected string
	Dim      string
	Error    string
	Success  string
	Warn     string
	Help     string
}

func defaultPalette() Palette {
	return Palette{
		Title:    "212",
		Header:   "39",
		Cursor:   "212",
		Selected: "212",
		Dim:      "240",
		Error:    "203",
		Success:  "42",
		Warn:     "214",
		Help:     "240",
	}
}

// paletteFromFile applies any color_* keys present in a parsed config file
// on top of the defaults.
func paletteFromFile(values map[string]string) Palette {
	p := defaultPalette()
	overrides := map[string]*string{
		"color_title":    &p.Title,
		"color_header":   &p.Header,
		"color_cursor":   &p.Cursor,
		"color_selected": &p.Selected,
		"color_dim":      &p.Dim,
		"color_error":    &p.Error,
		"color_success":  &p.Success,
		"color_warn":     &p.Warn,
		"color_help":     &p.Help,
	}
	for key, field := range overrides {
		if v, ok := values[key]; ok && v != "" {
			*field = v
		}
	}
	return p
}
