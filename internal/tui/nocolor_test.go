package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestApplyPalette_HonorsNoColor(t *testing.T) {
	tests := []struct {
		name      string
		env       string
		wantColor bool
	}{
		{"unset keeps palette colors", "", true},
		{"set disables colors", "1", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("NO_COLOR", tt.env)
			t.Cleanup(func() { ApplyPalette(DefaultPalette()) })
			ApplyPalette(DefaultPalette())

			_, isNone := titleStyle.GetForeground().(lipgloss.NoColor)
			if gotColor := !isNone; gotColor != tt.wantColor {
				t.Errorf("title has color = %v, want %v", gotColor, tt.wantColor)
			}
			_, borderNone := footerPanelStyle.GetBorderTopForeground().(lipgloss.NoColor)
			if gotColor := !borderNone; gotColor != tt.wantColor {
				t.Errorf("footer border has color = %v, want %v", gotColor, tt.wantColor)
			}
			if got := topBarStyles.Bar.GetReverse(); got == tt.wantColor {
				t.Errorf("top bar reverse = %v, want %v", got, !tt.wantColor)
			}
			if _, bg := topBarStyles.Bar.GetBackground().(lipgloss.NoColor); bg == tt.wantColor {
				t.Errorf("top bar background none = %v, want %v", bg, !tt.wantColor)
			}
		})
	}
}
