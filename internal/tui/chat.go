package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/morefun2602/opencode-go/internal/store"
)

type chatModel struct {
	viewport viewport.Model
	renderer *glamour.TermRenderer
	sticky   bool
	ready    bool
}

func newChat(theme Theme, width, height int) chatModel {
	style := "dark"
	if theme.Name == "light" {
		style = "light"
	}
	r, err := glamour.NewTermRenderer(glamour.WithStylePath(style), glamour.WithWordWrap(80))
	if err != nil {
		r = nil
	}

	vp := viewport.New(width, height)
	vp.MouseWheelEnabled = true

	return chatModel{
		viewport: vp,
		renderer: r,
		sticky:   true,
	}
}

func (c *chatModel) SetSize(w, h int) {
	c.viewport.Width = w
	c.viewport.Height = h
	c.ready = true
}

// Update forwards tea.Msg to the viewport so mouse wheel scrolling works.
func (c *chatModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	c.viewport, cmd = c.viewport.Update(msg)
	if c.viewport.AtBottom() {
		c.sticky = true
	} else if _, ok := msg.(tea.MouseMsg); ok {
		c.sticky = false
	}
	return cmd
}

// Refresh rebuilds all content from the messages list.
func (c *chatModel) Refresh(msgs []store.MessageRow, streamBuf string, theme Theme) {
	blocks := BuildRenderBlocks(msgs, theme, c.viewport.Width, c.renderer)
	content := BlocksToString(blocks)
	if streamBuf != "" {
		content += streamBuf + "\n"
	}
	c.viewport.SetContent(content)
	if c.sticky {
		c.viewport.GotoBottom()
	}
}

func (c *chatModel) View() string {
	if !c.ready {
		return ""
	}
	return c.viewport.View()
}

func (c *chatModel) ScrollUp(lines int) {
	c.sticky = false
	c.viewport.LineUp(lines)
}

func (c *chatModel) ScrollDown(lines int) {
	c.viewport.LineDown(lines)
	if c.viewport.AtBottom() {
		c.sticky = true
	}
}

func (c *chatModel) PageUp() {
	c.sticky = false
	c.viewport.HalfViewUp()
}

func (c *chatModel) PageDown() {
	c.viewport.HalfViewDown()
	if c.viewport.AtBottom() {
		c.sticky = true
	}
}

func (c *chatModel) GotoBottom() {
	c.sticky = true
	c.viewport.GotoBottom()
}

func (c *chatModel) ScrollPercent() string {
	return fmt.Sprintf("%d%%", int(c.viewport.ScrollPercent()*100))
}
