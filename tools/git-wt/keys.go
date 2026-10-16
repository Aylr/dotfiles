package main

import "charm.land/bubbles/v2/key"

type keyMap struct {
	Up, Down, Top, Bottom    key.Binding
	PageUp, PageDown         key.Binding
	Mark, Diff, Filter       key.Binding
	Delete, Help, Quit       key.Binding
	Confirm, Back            key.Binding
	Accept, Clear, ForceQuit key.Binding
	nav                      key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Top:       key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
		Bottom:    key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
		PageUp:    key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "scroll detail up")),
		PageDown:  key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "scroll detail down")),
		Mark:      key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "mark")),
		Diff:      key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "diff")),
		Filter:    key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Delete:    key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete marked")),
		Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Confirm:   key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "delete")),
		Back:      key.NewBinding(key.WithKeys("n", "esc", "q"), key.WithHelp("n/esc", "back")),
		Accept:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "keep filter")),
		Clear:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear filter")),
		ForceQuit: key.NewBinding(key.WithKeys("ctrl+c")),
		nav:       key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "move")),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.nav, k.Mark, k.Diff, k.Filter, k.Delete, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Top, k.Bottom},
		{k.Mark, k.Delete, k.Diff},
		{k.PageUp, k.PageDown},
		{k.Filter, k.Clear, k.Help, k.Quit},
	}
}

func (k keyMap) filterHelp() []key.Binding {
	return []key.Binding{k.nav, k.Accept, k.Clear}
}
