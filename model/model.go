package model

import (
	"clx/browser"
	"clx/cli"
	"clx/comment"
	"clx/constants/categories"
	"clx/constants/help"
	"clx/constants/messages"
	"clx/constants/panels"
	"clx/constants/state"
	"clx/core"
	"clx/file"
	"clx/retriever"
	"clx/screen"
	"clx/utils/message"
	"clx/utils/vim"
	"clx/view"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	constructor "clx/constructors"

	"github.com/gdamore/tcell/v2"
	"gitlab.com/tslocum/cview"
)

func SetAfterInitializationAndAfterResizeFunctions(app *cview.Application, list *cview.List,
	main *core.MainView, appState *core.ApplicationState, config *core.Config,
	ret *retriever.Retriever) {
	app.SetAfterResizeFunc(func(width int, height int) {
		if appState.IsReturningFromSuspension {
			appState.IsReturningFromSuspension = false

			return
		}

		resetStates(appState, ret)
		initializeView(appState, main, ret)

		listItems, err := ret.GetSubmissions(appState.CurrentCategory, appState.CurrentPage,
			appState.SubmissionsToShow, config.HighlightHeadlines, config.HideYCJobs)
		if err != nil {
			setToErrorState(appState, main, list, app)

			return
		}

		appState.State = state.OnSubmissionPage
		statusBarText := getInfoScreenStatusBarText(appState.CurrentHelpScreenCategory)
		marginText := getMarginText(config.RelativeNumbering, appState.SubmissionsToShow, len(listItems), 0, 0)

		view.ShowItems(list, listItems)
		view.SetLeftMarginText(main, marginText)

		if appState.State == state.OnHelpScreen {
			updateInfoScreenView(main, appState.CurrentHelpScreenCategory, statusBarText)
		}
	})
}

func setToErrorState(appState *core.ApplicationState, main *core.MainView, list *cview.List, app *cview.Application) {
	errorMessage := message.Error(messages.OfflineMessage)
	appState.State = state.Offline

	view.SetPermanentStatusBar(main, errorMessage, cview.AlignCenter)
	view.ClearList(list)
	app.Draw()
}

func resetStates(appState *core.ApplicationState, ret *retriever.Retriever) {
	resetApplicationState(appState)
	ret.Reset()
}

func resetApplicationState(appState *core.ApplicationState) {
	appState.CurrentPage = 0
	appState.ScreenWidth = screen.GetTerminalWidth()
	appState.ScreenHeight = screen.GetTerminalHeight()
	appState.SubmissionsToShow = screen.GetSubmissionsToShow(appState.ScreenHeight, 30)
}

func initializeView(appState *core.ApplicationState, main *core.MainView, ret *retriever.Retriever) {
	header := ret.GetHackerNewsHeader(appState.CurrentCategory)

	view.UpdateSettingsScreen(main)
	view.UpdateInfoScreen(main)
	view.SetPanelToSubmissions(main)
	view.SetHackerNewsHeader(main, header)
	view.SetPageCounter(main, appState.CurrentPage, ret.GetMaxPages(appState.CurrentCategory,
		appState.SubmissionsToShow))
}

func ReadSubmissionComments(app *cview.Application, main *core.MainView, list *cview.List,
	appState *core.ApplicationState, config *core.Config, r *retriever.Retriever) {
	story := r.GetStory(appState.CurrentCategory, list.GetCurrentItemIndex(), appState.SubmissionsToShow,
		appState.CurrentPage)

	app.Suspend(func() {
		id := strconv.Itoa(story.ID)

		comments, err := comment.FetchComments(id)
		if err != nil {
			errorMessage := message.Error(messages.CommentsNotFetched)
			view.SetTemporaryStatusBar(app, main, errorMessage, 4*time.Second)

			return
		}

		r.UpdateFavoriteStoryAndWriteToDisk(comments)
		screenWidth := screen.GetTerminalWidth()
		commentTree := comment.ToString(*comments, config.IndentSize, config.CommentWidth, screenWidth,
			config.PreserveRightMargin)

		cli.Less(commentTree)
	})

	changePage(app, list, main, appState, config, r, 0)
	appState.IsReturningFromSuspension = true
}

func OpenCommentsInBrowser(list *cview.List, appState *core.ApplicationState, r *retriever.Retriever) {
	story := r.GetStory(appState.CurrentCategory, list.GetCurrentItemIndex(), appState.SubmissionsToShow,
		appState.CurrentPage)
	url := "https://news.ycombinator.com/item?id=" + strconv.Itoa(story.ID)
	browser.Open(url)
}

func OpenLinkInBrowser(list *cview.List, appState *core.ApplicationState, r *retriever.Retriever) {
	story := r.GetStory(appState.CurrentCategory, list.GetCurrentItemIndex(), appState.SubmissionsToShow,
		appState.CurrentPage)
	browser.Open(story.URL)
}

func NextPage(app *cview.Application, list *cview.List, main *core.MainView, appState *core.ApplicationState,
	config *core.Config, ret *retriever.Retriever) {
	isOnLastPage := appState.CurrentPage+1 > ret.GetMaxPages(appState.CurrentCategory, appState.SubmissionsToShow)
	if isOnLastPage {
		return
	}

	changePage(app, list, main, appState, config, ret, 1)
}

func PreviousPage(app *cview.Application, list *cview.List, main *core.MainView, appState *core.ApplicationState,
	config *core.Config, ret *retriever.Retriever) {
	isOnFirstPage := appState.CurrentPage-1 < 0
	if isOnFirstPage {
		return
	}

	changePage(app, list, main, appState, config, ret, -1)
}

func changePage(app *cview.Application, list *cview.List, main *core.MainView, appState *core.ApplicationState,
	config *core.Config, ret *retriever.Retriever, delta int) {
	currentlySelectedItem := list.GetCurrentItemIndex()
	appState.CurrentPage += delta

	listItems, err := ret.GetSubmissions(appState.CurrentCategory, appState.CurrentPage,
		appState.SubmissionsToShow, config.HighlightHeadlines, config.HideYCJobs)
	if err != nil {
		setToErrorState(appState, main, list, app)

		return
	}

	view.ShowItems(list, listItems)
	view.SelectItem(list, currentlySelectedItem)

	ClearVimRegister(main, appState)

	marginText := getMarginText(config.RelativeNumbering, appState.SubmissionsToShow, len(listItems),
		list.GetCurrentItemIndex(), appState.CurrentPage)
	header := ret.GetHackerNewsHeader(appState.CurrentCategory)
	maxPages := ret.GetMaxPages(appState.CurrentCategory, appState.SubmissionsToShow)

	view.SetLeftMarginText(main, marginText)
	view.SetHackerNewsHeader(main, header)
	view.SetPageCounter(main, appState.CurrentPage, maxPages)
}

func getMarginText(useRelativeNumbering bool, viewableStories, maxItems, currentPosition, currentPage int) string {
	if maxItems == 0 {
		return ""
	}

	if useRelativeNumbering {
		return vim.RelativeRankings(viewableStories, maxItems, currentPosition, currentPage)
	}

	return vim.AbsoluteRankings(viewableStories, maxItems, currentPage)
}

func ChangeCategory(app *cview.Application, event *tcell.EventKey, list *cview.List, appState *core.ApplicationState,
	main *core.MainView, config *core.Config, ret *retriever.Retriever) {
	currentItem := list.GetCurrentItemIndex()
	nextCategory := 0

	if event.Key() == tcell.KeyBacktab {
		nextCategory = ret.GetPreviousCategory(appState.CurrentCategory)
	} else {
		nextCategory = ret.GetNextCategory(appState.CurrentCategory)
	}

	appState.CurrentCategory = nextCategory
	appState.CurrentPage = 0

	listItems, err := ret.GetSubmissions(appState.CurrentCategory, appState.CurrentPage,
		appState.SubmissionsToShow, config.HighlightHeadlines, config.HideYCJobs)
	if err != nil {
		setToErrorState(appState, main, list, app)

		return
	}

	view.ShowItems(list, listItems)
	view.SelectItem(list, currentItem)
	ClearVimRegister(main, appState)

	header := ret.GetHackerNewsHeader(appState.CurrentCategory)
	marginText := getMarginText(config.RelativeNumbering, appState.SubmissionsToShow, len(listItems),
		list.GetCurrentItemIndex(), appState.CurrentPage)
	maxPages := ret.GetMaxPages(appState.CurrentCategory, appState.SubmissionsToShow)

	view.SetLeftMarginText(main, marginText)
	view.SetPageCounter(main, appState.CurrentPage, maxPages)
	view.SetHackerNewsHeader(main, header)
}

func getNextCategory(currentCategory int, numberOfCategories int) int {
	if currentCategory == (numberOfCategories - 1) {
		return 0
	}

	return currentCategory + 1
}

func getPreviousCategory(currentCategory int, numberOfCategories int) int {
	if currentCategory == 0 {
		return numberOfCategories - 1
	}

	return currentCategory - 1
}

func ChangeHelpScreenCategory(event *tcell.EventKey, appState *core.ApplicationState, main *core.MainView) {
	if event.Key() == tcell.KeyBacktab {
		appState.CurrentHelpScreenCategory = getPreviousCategory(appState.CurrentHelpScreenCategory, 3)
	} else {
		appState.CurrentHelpScreenCategory = getNextCategory(appState.CurrentHelpScreenCategory, 3)
	}

	statusBarText := getInfoScreenStatusBarText(appState.CurrentHelpScreenCategory)

	updateInfoScreenView(main, appState.CurrentHelpScreenCategory, statusBarText)
}

func ShowCreateConfigConfirmationMessage(main *core.MainView, appState *core.ApplicationState) {
	if file.ConfigFileExists() {
		return
	}

	appState.IsOnConfigCreationConfirmationMessage = true

	view.SetPermanentStatusBar(main,
		"[::b]config.env[::-] will be created in [::r]~/.config/circumflex[::-], press Y to Confirm", cview.AlignCenter)
}

func ScrollSettingsOneLineUp(main *core.MainView) {
	view.ScrollSettingsOneLineUp(main)
}

func ScrollSettingsOneLineDown(main *core.MainView) {
	view.ScrollSettingsOneLineDown(main)
}

func ScrollSettingsOneHalfPageUp(main *core.MainView) {
	halfPage := screen.GetTerminalHeight() / 2
	view.ScrollSettingsByAmount(main, -halfPage)
}

func ScrollSettingsOneHalfPageDown(main *core.MainView) {
	halfPage := screen.GetTerminalHeight() / 2
	view.ScrollSettingsByAmount(main, halfPage)
}

func ScrollSettingsToBeginning(main *core.MainView) {
	view.ScrollSettingsToBeginning(main)
}

func ScrollSettingsToEnd(main *core.MainView) {
	view.ScrollSettingsToEnd(main)
}

func CancelConfirmation(appState *core.ApplicationState, main *core.MainView) {
	appState.IsOnAddFavoriteConfirmationMessage = false
	appState.IsOnDeleteFavoriteConfirmationMessage = false
	appState.IsOnConfigCreationConfirmationMessage = false

	view.SetPermanentStatusBar(main, "Cancelled", cview.AlignCenter)
}

func CreateConfig(appState *core.ApplicationState, main *core.MainView) {
	statusBarMessage := ""
	appState.IsOnConfigCreationConfirmationMessage = false

	err := file.WriteToFile(file.PathToConfigFile(), constructor.GetConfigFileContents())
	if err != nil {
		statusBarMessage = message.Error(messages.ConfigNotCreated)
	} else {
		statusBarMessage = message.Success(messages.ConfigCreatedAt)
	}

	view.UpdateSettingsScreen(main)
	view.SetPermanentStatusBar(main, statusBarMessage, cview.AlignCenter)
}

func SelectItemDown(main *core.MainView, list *cview.List, appState *core.ApplicationState, config *core.Config) {
	currentItem := list.GetCurrentItemIndex()
	itemCount := list.GetItemCount()
	nextItem := vim.GetItemDown(appState.VimNumberRegister, currentItem, itemCount)

	view.SelectItem(list, nextItem)

	ClearVimRegister(main, appState)
	marginText := getMarginText(config.RelativeNumbering, appState.SubmissionsToShow, list.GetItemCount(), nextItem,
		appState.CurrentPage)
	view.SetLeftMarginText(main, marginText)
	view.ClearStatusBar(main)
}

func SelectItemUp(main *core.MainView, list *cview.List, appState *core.ApplicationState, config *core.Config) {
	currentItem := list.GetCurrentItemIndex()
	nextItem := vim.GetItemUp(appState.VimNumberRegister, currentItem)

	view.SelectItem(list, nextItem)

	ClearVimRegister(main, appState)
	marginText := getMarginText(config.RelativeNumbering, appState.SubmissionsToShow, list.GetItemCount(), nextItem,
		appState.CurrentPage)
	view.SetLeftMarginText(main, marginText)
	view.ClearStatusBar(main)
}

func EnterInfoScreen(main *core.MainView, appState *core.ApplicationState) {
	statusBarText := getInfoScreenStatusBarText(appState.CurrentHelpScreenCategory)
	appState.State = state.OnHelpScreen

	ClearVimRegister(main, appState)
	updateInfoScreenView(main, appState.CurrentHelpScreenCategory, statusBarText)
}

func getInfoScreenStatusBarText(category int) string {
	if category == help.Info {
		return messages.GetCircumflexStatusMessage()
	}

	return ""
}

func updateInfoScreenView(main *core.MainView, helpScreenCategory int, statusBarText string) {
	view.SetPermanentStatusBar(main, statusBarText, cview.AlignCenter)
	view.HidePageCounter(main)
	view.SetHelpScreenHeader(main, helpScreenCategory)
	view.HideLeftMarginRanks(main)
	view.SetHelpScreenPanel(main, helpScreenCategory)
}

func ExitHelpScreen(main *core.MainView, appState *core.ApplicationState, config *core.Config, list *cview.List,
	ret *retriever.Retriever) {
	appState.State = state.OnSubmissionPage

	marginText := getMarginText(config.RelativeNumbering, appState.SubmissionsToShow, list.GetItemCount(),
		list.GetCurrentItemIndex(), appState.CurrentPage)
	header := ret.GetHackerNewsHeader(appState.CurrentCategory)
	maxPages := ret.GetMaxPages(appState.CurrentCategory, appState.SubmissionsToShow)

	view.SetLeftMarginText(main, marginText)
	view.SetHackerNewsHeader(main, header)
	view.SetPanelToSubmissions(main)
	view.SetPageCounter(main, appState.CurrentPage, maxPages)
	view.ClearStatusBar(main)
}

func SelectFirstElementInList(main *core.MainView, appState *core.ApplicationState, list *cview.List,
	config *core.Config) {
	view.SelectFirstElementInList(list)
	ClearVimRegister(main, appState)

	marginText := getMarginText(config.RelativeNumbering, appState.SubmissionsToShow, list.GetItemCount(),
		list.GetCurrentItemIndex(), appState.CurrentPage)
	view.SetLeftMarginText(main, marginText)
}

func GoToLowerCaseG(main *core.MainView, appState *core.ApplicationState, list *cview.List, config *core.Config) {
	switch {
	case appState.VimNumberRegister == "g":
		SelectFirstElementInList(main, appState, list, config)

		marginText := getMarginText(config.RelativeNumbering, appState.SubmissionsToShow, list.GetItemCount(),
			list.GetCurrentItemIndex(),
			appState.CurrentPage)

		view.SetLeftMarginText(main, marginText)
		view.ClearStatusBar(main)

	case vim.ContainsOnlyNumbers(appState.VimNumberRegister):
		appState.VimNumberRegister += "g"

		view.SetPermanentStatusBar(main, vim.FormatRegisterOutput(appState.VimNumberRegister), cview.AlignRight)

	case vim.IsNumberWithGAppended(appState.VimNumberRegister):
		register := strings.TrimSuffix(appState.VimNumberRegister, "g")

		itemToJumpTo := vim.GetItemToJumpTo(register,
			list.GetCurrentItemIndex(),
			appState.SubmissionsToShow,
			appState.CurrentPage)

		ClearVimRegister(main, appState)
		view.SelectItem(list, itemToJumpTo)
		view.ClearStatusBar(main)

		marginText := getMarginText(config.RelativeNumbering, appState.SubmissionsToShow, list.GetItemCount(),
			list.GetCurrentItemIndex(), appState.CurrentPage)
		view.SetLeftMarginText(main, marginText)

	case appState.VimNumberRegister == "":
		appState.VimNumberRegister += "g"

		view.SetPermanentStatusBar(main, vim.FormatRegisterOutput(appState.VimNumberRegister), cview.AlignRight)
	}
}

func GoToUpperCaseG(main *core.MainView, appState *core.ApplicationState, list *cview.List, config *core.Config) {
	switch {
	case appState.VimNumberRegister == "":
		view.SelectLastElementInList(list)
		ClearVimRegister(main, appState)

		marginText := getMarginText(config.RelativeNumbering, appState.SubmissionsToShow, list.GetItemCount(),
			list.GetCurrentItemIndex(), appState.CurrentPage)
		view.SetLeftMarginText(main, marginText)

	case vim.ContainsOnlyNumbers(appState.VimNumberRegister):
		register := strings.TrimSuffix(appState.VimNumberRegister, "g")

		itemToJumpTo := vim.GetItemToJumpTo(register, list.GetCurrentItemIndex(), appState.SubmissionsToShow,
			appState.CurrentPage)

		ClearVimRegister(main, appState)
		view.SelectItem(list, itemToJumpTo)
		view.ClearStatusBar(main)

		marginText := getMarginText(config.RelativeNumbering, appState.SubmissionsToShow, list.GetItemCount(),
			list.GetCurrentItemIndex(), appState.CurrentPage)
		view.SetLeftMarginText(main, marginText)
	case vim.IsNumberWithGAppended(appState.VimNumberRegister):
		ClearVimRegister(main, appState)
		view.ClearStatusBar(main)

	case appState.VimNumberRegister == "g":
		ClearVimRegister(main, appState)
		view.ClearStatusBar(main)
	}
}

func PutDigitInRegister(main *core.MainView, element rune, appState *core.ApplicationState) {
	if len(appState.VimNumberRegister) == 0 && string(element) == "0" {
		return
	}

	if appState.VimNumberRegister == "g" {
		ClearVimRegister(main, appState)
	}

	registerIsMoreThanThreeDigits := len(appState.VimNumberRegister) > 2

	if registerIsMoreThanThreeDigits {
		appState.VimNumberRegister = trimFirstRune(appState.VimNumberRegister)
	}

	appState.VimNumberRegister += string(element)

	view.SetPermanentStatusBar(main, vim.FormatRegisterOutput(appState.VimNumberRegister), cview.AlignRight)
}

func trimFirstRune(s string) string {
	_, i := utf8.DecodeRuneInString(s)

	return s[i:]
}

func Quit(app *cview.Application) {
	app.Stop()
}

func ClearVimRegister(main *core.MainView, appState *core.ApplicationState) {
	appState.VimNumberRegister = ""

	view.ClearStatusBar(main)
}

func Refresh(app *cview.Application, list *cview.List, main *core.MainView, appState *core.ApplicationState,
	config *core.Config, ret *retriever.Retriever) {
	afterResizeFunc := app.GetAfterResizeFunc()
	afterResizeFunc(appState.ScreenWidth, appState.ScreenHeight)

	ExitHelpScreen(main, appState, config, list, ret)

	if appState.State == state.Offline {
		errorMessage := message.Error(messages.OfflineMessage)

		view.SetPermanentStatusBar(main, errorMessage, cview.AlignCenter)
		view.ClearList(list)
		app.Draw()
	} else {
		duration := time.Millisecond * 2000
		view.SetTemporaryStatusBar(app, main, "Refreshed", duration)
	}
}

func AddToFavoritesConfirmationDialogue(main *core.MainView, appState *core.ApplicationState, list *cview.List) {
	if list.GetItemCount() == 0 {
		return
	}

	appState.IsOnAddFavoriteConfirmationMessage = true

	view.SetPermanentStatusBar(main, "[green]Add[-] to Favorites? Press [::b]Y[::-] to Confirm", cview.AlignCenter)
}

func DeleteFavoriteConfirmationDialogue(main *core.MainView, appState *core.ApplicationState, list *cview.List) {
	if list.GetItemCount() == 0 {
		return
	}

	appState.IsOnDeleteFavoriteConfirmationMessage = true

	view.SetPermanentStatusBar(main,
		"[red]Delete[-] from Favorites? Press [::b]Y[::-] to Confirm", cview.AlignCenter)
}

func AddToFavorites(app *cview.Application, list *cview.List, main *core.MainView, appState *core.ApplicationState,
	config *core.Config, ret *retriever.Retriever) {
	statusBarMessage := ""
	appState.IsOnAddFavoriteConfirmationMessage = false

	story := ret.GetStory(appState.CurrentCategory, list.GetCurrentItemIndex(), appState.SubmissionsToShow,
		appState.CurrentPage)
	ret.AddItemToFavorites(story)
	bytes, _ := ret.GetFavoritesJSON()
	filePath := file.PathToFavoritesFile()

	err := file.WriteToFile(filePath, string(bytes))
	if err != nil {
		statusBarMessage = message.Error("Could not add to favorites")
	} else {
		statusBarMessage = message.Success("Item added to favorites")
	}

	changePage(app, list, main, appState, config, ret, 0)
	view.SetPermanentStatusBar(main, statusBarMessage, cview.AlignCenter)
}

func DeleteItem(app *cview.Application, list *cview.List, appState *core.ApplicationState,
	main *core.MainView, config *core.Config, ret *retriever.Retriever) {
	appState.IsOnDeleteFavoriteConfirmationMessage = false
	ret.DeleteStoryAndWriteToDisk(appState.CurrentCategory, list.GetCurrentItemIndex(), appState.SubmissionsToShow,
		appState.CurrentPage)

	hasDeletedLastItemOnSecondOrThirdPage := list.GetCurrentItemIndex() == 0 &&
		list.GetItemCount() == 1 && appState.CurrentPage != 0
	hasDeletedLastItemOnFirstPage := list.GetCurrentItemIndex() == 0 &&
		list.GetItemCount() == 1 && appState.CurrentPage == 0

	switch {
	case hasDeletedLastItemOnSecondOrThirdPage:
		changePage(app, list, main, appState, config, ret, -1)
	case hasDeletedLastItemOnFirstPage:
		appState.CurrentCategory = categories.Show
		ChangeCategory(app, tcell.NewEventKey(tcell.KeyTab, ' ', tcell.ModNone), list, appState, main, config, ret)
	default:
		changePage(app, list, main, appState, config, ret, 0)
	}

	m := message.Success("Item deleted")
	view.SetPermanentStatusBar(main, m, cview.AlignCenter)
}

func ShowAddCustomFavorite(app *cview.Application, list *cview.List, main *core.MainView,
	appState *core.ApplicationState, config *core.Config, ret *retriever.Retriever) {
	appState.IsOnAddFavoriteByID = true

	view.HideLeftMarginRanks(main)

	main.CustomFavorite.SetText("")
	main.CustomFavorite.SetAcceptanceFunc(cview.InputFieldInteger)
	main.CustomFavorite.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			appState.IsOnAddFavoriteByID = false
			text := main.CustomFavorite.GetText()
			id, _ := strconv.Atoi(text)

			item := new(core.Submission)
			item.ID = id
			item.Title = "[Enter comment section to update fields]"
			item.Time = time.Now().Unix()

			ret.AddItemToFavorites(item)
			bytes, _ := ret.GetFavoritesJSON()
			filePath := file.PathToFavoritesFile()

			_ = file.WriteToFile(filePath, string(bytes))
		}

		main.Panels.SetCurrentPanel(panels.SubmissionsPanel)
		app.SetFocus(main.Grid)

		changePage(app, list, main, appState, config, ret, 0)
		appState.IsOnAddFavoriteByID = false
	})

	app.SetFocus(main.CustomFavorite)

	view.ShowFavoritesBox(main)
}
