package main

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// The window is a darker gutter with rounded "islands" floating on it: the
// sidebar and the workspace each sit on their own panel. Inside them,
// controls are flat: a tint under the pointer or on the chosen row stands
// in for borders.

const (
	islandGap     = 10
	sidebarWidth  = 248
	islandRadius  = 14
	timelineWidth = 760 // the conversation and its composer are this wide at most
)

var tintFade = ui.Animation{Duration: 120 * time.Millisecond}

// gutterColor fills the window between and around the islands.
func gutterColor(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#0d0d0f")
	}
	return ui.Hex("#e8e9ee")
}

// island turns e into a panel: rounded, clipped to its corners, hairline
// border and a soft shadow in the light appearance.
func island(e ui.Element, t *ui.Theme) ui.Element {
	e = e.Radius(islandRadius).Clip().BorderWidth(1).BorderColor(t.Border.Alpha(0.55))
	if !t.Dark {
		e = e.Shadow(0, 1, 4, 0, ui.RGBA(0, 0, 0, 0.06))
	}
	return e
}

// card is a smaller rounded surface inside an island, for the composer,
// tool calls and plans.
func card(e ui.Element, t *ui.Theme) ui.Element {
	return e.Background(t.Surface).Radius(12)
}

// navButton is a full-width row of text without a border: tinted under
// the pointer, and in the accent where it is the chosen one.
func navButton(c *ui.Context, icon *ui.SVG, text string, selected bool) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Label(text).Height(34).Padding(0, 10).Gap(8).Radius(9).AlignItems(ui.Center).Justify(ui.Start)
	bg, fg := ui.Transparent, t.Text
	switch {
	case selected:
		bg, fg = t.Accent.Alpha(0.12), t.Accent
	case b.Hovered():
		bg = t.Text.Alpha(0.06)
	}
	b.Background(b.AnimateColor("bg", bg, tintFade)).TextColor(fg)
	b.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(16, 16)
		}
		ui.Text(c, text).FontSize(13).SingleLine()
	})
	return b
}

// ghostButton is a small button that is only text until the pointer is
// over it. Label names it for assistive tools and tests.
func ghostButton(c *ui.Context, text, label string) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Label(label).Height(26).Padding(0, 10).Radius(8).AlignItems(ui.Center).TextColor(t.TextMuted)
	bg := ui.Transparent
	if b.Hovered() {
		bg = t.Text.Alpha(0.08)
		b.TextColor(t.Text)
	}
	b.Background(b.AnimateColor("bg", bg, tintFade))
	b.Children(func() { ui.Text(c, text).FontSize(12).SingleLine() })
	return b
}

// centered lays fn out in a column that is at most maxWidth wide, in the
// middle of the available width.
func centered(c *ui.Context, maxWidth float32, fn func()) {
	ui.Row(c).Justify(ui.Center).Padding(0, 28).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).MaxWidth(maxWidth).Children(fn)
	})
}

// joinDetail joins the parts that are not empty with a dot between.
func joinDetail(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, " · ")
}

// workspaceLeft is the room left of the workspace island: none against the
// sidebar, a gutter's worth where the sidebar is hidden.
func workspaceLeft(sidebarHidden bool) float32 {
	if sidebarHidden {
		return islandGap
	}
	return 0
}
