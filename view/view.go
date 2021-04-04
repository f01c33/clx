package view

import (
	"clx/constants/panels"
	"clx/core"
	"clx/header"
	"clx/pages"
	"time"

	"gitlab.com/tslocum/cview"
)

func SetHackerNewsHeader(m *core.MainView, header string) {
	m.Header.SetText(header)
}

func SetHelpScreenHeader(m *core.MainView, category int) {
	h := header.GetCircumflexHeader(category)
	m.Header.SetText(h)
}

func SetPanelToMainView(m *core.MainView) {
	m.Panels.SetCurrentPanel(panels.SubmissionsPanel)
}

func SetPanelToInfoView(m *core.MainView) {
	m.Panels.SetCurrentPanel(panels.InfoPanel)
	m.InfoScreen.ScrollToBeginning()
}

func ClearStatusBar(m *core.MainView) {
	SetPermanentStatusBar(m, "", cview.AlignCenter)
}

func SetPermanentStatusBar(m *core.MainView, text string, align int) {
	m.StatusBar.SetTextAlign(align)
	m.StatusBar.SetText(text)
}

func SetTemporaryStatusBar(app *cview.Application, m *core.MainView, text string, duration time.Duration) {
	go setAndClearStatusBar(app, m, text, duration)
}

func setAndClearStatusBar(app *cview.Application, m *core.MainView, text string, duration time.Duration) {
	m.StatusBar.SetText(text)
	time.Sleep(duration)
	m.StatusBar.SetText("")
	app.Draw()
}

func SetLeftMarginText(m *core.MainView, text string) {
	m.LeftMargin.SetText(text)
}

func HideLeftMarginRanks(m *core.MainView) {
	m.LeftMargin.SetText("")
}

func HidePageCounter(m *core.MainView) {
	m.PageCounter.SetText("")
}

func ScrollInfoScreenByAmount(m *core.MainView, amount int) {
	row, col := m.InfoScreen.GetScrollOffset()
	m.InfoScreen.ScrollTo(row+amount, col)
}

func ScrollInfoScreenToBeginning(m *core.MainView) {
	m.InfoScreen.ScrollToBeginning()
}

func ScrollInfoScreenToEnd(m *core.MainView) {
	m.InfoScreen.ScrollToEnd()
}

func SetPageCounter(m *core.MainView, currentPage int, maxPages int) {
	pageCounter := pages.GetPageCounter(currentPage, maxPages)
	m.PageCounter.SetText(pageCounter)
}

func SelectFirstElementInList(list *cview.List) {
	firstElement := 0
	list.SetCurrentItem(firstElement)
}

func SelectLastElementInList(list *cview.List) {
	lastElement := -1
	list.SetCurrentItem(lastElement)
}

func ClearList(list *cview.List) {
	list.Clear()
}

func SelectItem(list *cview.List, index int) {
	list.SetCurrentItem(index)
}

func ShowItems(list *cview.List, listItems []*cview.ListItem) {
	list.Clear()

	for _, item := range listItems {
		list.AddItem(item)
	}
}

func ShowFavoritesBox(m *core.MainView) {
	m.Panels.SetCurrentPanel(panels.AddCustomFavoritePanel)
}

func SetInfoScreenText(m *core.MainView, text string) {
	m.InfoScreen.SetText(text)
}
