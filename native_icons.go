package main

import "github.com/egoist/mygo/ui"

// Icons drawn on a 24-point grid in a 1.75-point stroke, in the color of
// the text around them.

func strokeIcon(paths string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">` + paths + `</svg>`))
}

var (
	iconClose    = strokeIcon(`<path d="M18 6 6 18"/><path d="m6 6 12 12"/>`)
	iconSidebar  = strokeIcon(`<rect x="3" y="4" width="18" height="16" rx="3"/><path d="M9.5 4v16"/>`)
	iconSettings = strokeIcon(`<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"/>`)
	iconBack     = strokeIcon(`<path d="m15 18-6-6 6-6"/>`)
	iconTrash    = strokeIcon(`<path d="M3 6h18"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/><path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>`)
	iconRefresh  = strokeIcon(`<path d="M21 12a9 9 0 1 1-3-6.7L21 8"/><path d="M21 3v5h-5"/>`)
)

// iconButton is a borderless button showing an icon, tinted under the
// pointer. Label names it for assistive tools and tests.
func iconButton(c *ui.Context, icon *ui.SVG, size float32, label string) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Label(label).Size(size+10, size+10).Center().Radius(8).TextColor(t.TextMuted)
	bg := ui.Transparent
	if b.Hovered() {
		bg = t.Text.Alpha(0.09)
		b.TextColor(t.Text)
	}
	b.Background(b.AnimateColor("bg", bg, tintFade))
	b.Children(func() { ui.Icon(c, icon).Size(size, size) })
	return b
}
