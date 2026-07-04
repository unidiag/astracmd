package dashboard

import (
	"context"

	"main/internal/astra"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	PageMain   = "main"
	PageDialog = "dialog"
	PageError  = "error"
)

type FunctionKeyAction struct {
	Label  string
	Handle func()
}

// ConfigStore is a small adapter around the application config.
//
// The dashboard package should not import package main, so all operations
// related to the root config must be provided through this interface.
type ConfigStore interface {
	Save() error
	UpdateConnectionDebug(id string, enabled bool)
	ServiceProvider() string
}

// Options contains everything the dashboard package needs from the root UI.
//
// This keeps internal/dashboard independent from package main and allows the
// dashboard files to be moved gradually without dragging the whole UI type
// into the package.
type Options struct {
	App        *tview.Application
	Pages      *tview.Pages
	Connection astra.Connection
	Config     ConfigStore

	SetMain func(root tview.Primitive)

	ShowError       func(message string, returnFocus tview.Primitive)
	ShowHelp        func()
	ShowConnections func()
	Quit            func()

	StopDashboardTimer func()
	SetDashboardCancel func(cancel context.CancelFunc)

	HandleGlobalKeys func(event *tcell.EventKey) bool

	NewFunctionKeyBar func(actions map[int]FunctionKeyAction) tview.Primitive
}

func (opt Options) QueueUpdateDraw(fn func()) {
	if opt.App == nil || fn == nil {
		return
	}

	opt.App.QueueUpdateDraw(fn)
}

func (opt Options) SetFocus(p tview.Primitive) {
	if opt.App == nil || p == nil {
		return
	}

	opt.App.SetFocus(p)
}

func (opt Options) RemoveDialog() {
	if opt.Pages == nil {
		return
	}

	opt.Pages.RemovePage(PageDialog)
}

func (opt Options) RemoveError() {
	if opt.Pages == nil {
		return
	}

	opt.Pages.RemovePage(PageError)
}

func (opt Options) AddDialog(p tview.Primitive, focus bool) {
	if opt.Pages == nil || p == nil {
		return
	}

	opt.Pages.AddPage(PageDialog, p, false, focus)

	if focus {
		opt.SetFocus(p)
	}
}

func (opt Options) AddError(p tview.Primitive, focus bool) {
	if opt.Pages == nil || p == nil {
		return
	}

	opt.Pages.AddPage(PageError, p, false, focus)

	if focus {
		opt.SetFocus(p)
	}
}

func (opt Options) ServiceProvider() string {
	if opt.Config == nil {
		return ""
	}

	return opt.Config.ServiceProvider()
}
