package main

import (
	"errors"
	"log"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	x_widget "fyne.io/x/fyne/widget"

	"fyne.io/fyne/v2/dialog"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/Hashy-Software/hasherino-go/components"
	"github.com/Hashy-Software/hasherino-go/hasherino"
)

var (
	callbackMap      = make(map[string]func(hasherino.ChatMessage))
	defaultEmoteSize = fyne.NewSize(45, 45)
)

func NewSettingsTabs(hc *hasherino.HasherinoController, w fyne.Window) *container.AppTabs {
	// Accounts tab
	accounts, err := hc.GetAccounts()
	if err != nil {
		panic(err)
	}

	nCols := 3

	table := widget.NewTableWithHeaders(
		func() (int, int) {
			return len(accounts), nCols
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			account := accounts[i.Row]
			cols := []string{
				account.Id,
				account.Login,
				"",
			}
			if account.Active {
				cols[2] = "Yes"
			} else {
				cols[2] = "No"
			}
			o.(*widget.Label).SetText(cols[i.Col])
		},
	)
	table.UpdateHeader = func(id widget.TableCellID, o fyne.CanvasObject) {
		switch id.Col {
		case 0:
			o.(*widget.Label).SetText("ID")
		case 1:
			o.(*widget.Label).SetText("Login")
		case 2:
			o.(*widget.Label).SetText("Active")
		}
	}
	var selectedAccount *hasherino.Account
	table.OnSelected = func(id widget.TableCellID) {
		if id.Row >= 0 {
			selectedAccount = accounts[id.Row]
		}
	}

	accountsBox := container.NewBorder(
		nil,
		container.NewHBox(
			widget.NewButton("Add", func() {
				hc.OpenOAuthPage()
			}),
			widget.NewButton("Remove", func() {
				if selectedAccount != nil {
					hc.RemoveAccount(selectedAccount.Id)
					accounts, err = hc.GetAccounts()
					if err != nil {
						log.Println(err)
					}
					table.Refresh()
				}
			}),
			widget.NewButton("Activate", func() {
				if selectedAccount == nil {
					dialog.ShowError(errors.New("No account selected"), w)
					return
				}
				hc.SetActiveAccount(selectedAccount.Id)
				accounts, err = hc.GetAccounts()
				if err != nil {
					log.Println(err)
				}
				table.Refresh()
			}),
			widget.NewButton("Refresh", func() {
				accounts, err = hc.GetAccounts()
				if err != nil {
					log.Println(err)
				}
				table.Refresh()
			}),
		),
		nil,
		nil,
		table,
	)
	accountsBox.Add(table)

	// General tab
	chatLimitEntry := widget.NewEntry()
	settings, err := hc.GetSettings()
	if err != nil {
		panic(err)
	}
	chatLimitEntry.SetText(strconv.Itoa(settings.ChatMessageLimit))
	chatLimitEntry.Validator = func(s string) error {
		_, err := strconv.Atoi(s)
		return err
	}
	chatLimitEntry.OnChanged = func(s string) {
		settings.ChatMessageLimit, err = strconv.Atoi(s)
		if err != nil {
			log.Println(err)
		}
		err = hc.SetSettings(settings)
		if err != nil {
			log.Println(err)
		}
	}
	RobottyURL, err := url.Parse("https://recent-messages.robotty.de/")
	if err != nil {
		log.Printf("Could not parse Robotty URL: %v", err)
	}
	disclaimer := `
	This feature loads data from a third-party service on Startup. 
	Channels you join will be sent to that service, and the service will
	store messages for channels you visit to provice the service.
	Would you like to enable this feature?
	`
	historyChoice := widget.NewCheck("", func(b bool) {})
	historyChoice.OnChanged = func(b bool) {
		settings.ChatHistory = b
		if !b {
			settings.ChatHistory = false
			err = hc.SetSettings(settings)
			if err != nil {
				dialog.ShowError(err, w)
			}
			return
		} else {
			historyChoice.Checked = false // if the user clicks cancel, it has to remain unchecked
			dialog.ShowCustomConfirm(
				"Disclaimer",
				"",
				"Cancel",
				container.NewVBox(
					widget.NewLabel(disclaimer),
					widget.NewHyperlink("Click here for more information", RobottyURL),
				),
				func(b bool) {
					settings.ChatHistory = b
					historyChoice.Checked = b
					historyChoice.Refresh()
					err = hc.SetSettings(settings)
					if err != nil {
						dialog.ShowError(err, w)
					}
				},
				w,
			)
		}
	}
	historyChoice.Checked = settings.ChatHistory
	generalBox := container.NewVBox(
		container.NewHBox(widget.NewLabel("Chat message limit"), layout.NewSpacer(), chatLimitEntry),
		container.NewHBox(widget.NewLabel("Chat history"), layout.NewSpacer(), historyChoice),
		widget.NewLabel(""),
		widget.NewLabel(""),
		widget.NewLabel(""),
	)

	// Tabs
	tabs := container.NewAppTabs(
		container.NewTabItem("General", generalBox),
		container.NewTabItem("Accounts", accountsBox),
	)
	return tabs
}

func NewEmoteCanvasObject(emote *hasherino.Emote) (*fyne.CanvasObject, error) {
	url, err := emote.GetUrl()
	if err != nil {
		return nil, err
	}
	uri, err := storage.ParseURI(url)
	if err != nil {
		return nil, err
	}
	var imgContainer fyne.CanvasObject
	if emote.Animated {
		image, err := x_widget.NewAnimatedGif(uri)
		if err != nil {
			return nil, err
		}
		image.Start()
		imgContainer = container.NewWithoutLayout(image)
		image.Resize(defaultEmoteSize)
		imgContainer.Resize(defaultEmoteSize)
	} else {
		image := canvas.NewImageFromURI(uri)
		imgContainer = container.NewWithoutLayout(image)
		image.Resize(defaultEmoteSize)
		imgContainer.Resize(defaultEmoteSize)
	}

	return &imgContainer, nil
}

func NewChatTab(
	channel string,
	sendMsg func(string) (string, error),
	getEmotes func(string) ([]*hasherino.Emote, error),
	window fyne.Window,
	settingsFunc func() (*hasherino.AppSettings, error),
) *container.TabItem {
	var data []string = []string{}
	messageList := widget.NewList(
		func() int {
			return len(data)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("template")
			label.Wrapping = fyne.TextWrapWord
			return label
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(data[i])
		})
	callbackMap[channel] = func(message hasherino.ChatMessage) {
		if message.Command != "PRIVMSG" {
			return
		}
		text := message.Author + ": " + message.Text

		settings, err := settingsFunc()
		if err != nil {
			log.Println(err)
			return
		}
		if len(data) >= settings.ChatMessageLimit {
			data = append(data[1:], text)
		} else {
			data = append(data, text)
		}
		messageList.ScrollToBottom()
		messageList.Refresh()
	}
	go func() {
		settings, err := settingsFunc()
		if err != nil {
			log.Println(err)
			return
		}
		if !settings.ChatHistory {
			return
		}
		historyMsgs, err := hasherino.GetChatHistory(channel, settings.ChatMessageLimit)
		if err != nil {
			log.Println(err)
			return
		}
		callback, ok := callbackMap[channel]
		if !ok {
			log.Printf("No callback for channel %s.", channel)
			return
		}
		for _, msg := range *historyMsgs {
			callback(msg)
		}
	}()
	msgEntry := widget.NewEntry()
	msgEntry.SetPlaceHolder("Message")
	msgEntry.Validator = func(s string) error {
		if len(s) > 500 {
			return errors.New("Message too long")
		}
		return nil
	}
	msgEntry.OnSubmitted = func(text string) {
		err := msgEntry.Validate()
		if err != nil {
			dialog.ShowError(err, window)
			return
		}

		author, err := sendMsg(text)
		if err != nil {
			dialog.ShowError(err, window)
			return
		}

		msgEntry.SetText("")
		data = append(data, author+": "+text)
		messageList.ScrollToBottom()
		messageList.Refresh()
	}
	content := container.NewBorder(nil, container.NewBorder(nil, nil, nil, widget.NewButton("😃", func() {
		newWindow := fyne.CurrentApp().NewWindow("Select emote")
		newWindow.Resize(fyne.NewSize(300, 600))
		newWindow.SetContent(container.NewCenter(widget.NewLabel("Loading...")))

		loadEmoteSearch := func(search string) (*widget.Accordion, error) {
			emotes, err := getEmotes(search)
			if err != nil {
				dialog.ShowError(err, window)
				return nil, err
			}

			var images []fyne.CanvasObject
			var animatedEmotes []*components.EmoteGif
			mutex := sync.Mutex{}

			fourth := len(emotes) / 4
			emoteSlices := [][]*hasherino.Emote{
				emotes[:fourth],
				emotes[fourth : 2*fourth],
				emotes[2*fourth : 3*fourth],
				emotes[3*fourth:],
			}
			var wg sync.WaitGroup

			for _, emoteSlice := range emoteSlices {
				wg.Add(1)
				go func(emoteSlice []*hasherino.Emote) {
					defer wg.Done()

					for _, emote := range emoteSlice {
						imgCanvas, err := components.NewEmote(emote, func(text string) error {
							msgEntry.SetText(msgEntry.Text + text + " ")
							newWindow.Close()
							return nil
						})
						if err != nil {
							log.Println(err)
							continue
						}
						mutex.Lock()
						images = append(images, imgCanvas)
						if emote.Animated {
							animatedEmotes = append(animatedEmotes, imgCanvas.(*components.EmoteGif))
						}
						mutex.Unlock()
					}
				}(emoteSlice)
			}
			newWindow.SetOnClosed(func() {
				for _, emote := range animatedEmotes {
					go func(emote *components.EmoteGif) {
						emote.Stop()
					}(emote)
				}
			})
			wg.Wait()
			grid := container.NewGridWrap(defaultEmoteSize, images...)
			stvScroll := container.NewScroll(grid)
			stvScroll.OnScrolled = func(scrollOffset fyne.Position) {
				for _, comp := range images {
					go func(comp fyne.CanvasObject) {
						w := comp.(components.LazyLoadedWidget)
						scrollSize := stvScroll.Size()
						widgetPos := w.Position()
						widgetSize := w.Size()
						isVisible := widgetPos.Y+widgetSize.Height > scrollOffset.Y &&
							widgetPos.Y < scrollOffset.Y+scrollSize.Height
						if isVisible {
							w.LazyLoad()
						} else {
							w.LazyUnload()
						}
					}(comp)
				}
			}
			// Load the first 60 images
			// TODO: work for each accordion item
			start, end := 0, min(60, len(images))
			for i := start; i < end; i++ {
				go func(i int) {
					images[i].(components.LazyLoadedWidget).LazyLoad()
				}(i)
			}
			accordion := widget.NewAccordion(
				widget.NewAccordionItem("Twitch Emotes", widget.NewLabel("Not implemented")),
				widget.NewAccordionItem("7TV Emotes"+strings.Repeat(" ", 80), stvScroll),
				widget.NewAccordionItem("FFZ Emotes", widget.NewLabel("Not implemented")),
				widget.NewAccordionItem("BTTV Emotes", widget.NewLabel("Not implemented")),
				widget.NewAccordionItem("Emoji", widget.NewLabel("Not implemented")),
			)
			accordion.Items[1].Open = true
			return accordion, nil
		}

		searchEntry := widget.NewEntry()
		searchEntry.SetPlaceHolder("Emote name")
		searchEntry.OnChanged = func(s string) {
			accordion, err := loadEmoteSearch(s)
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			newWindow.SetContent(container.NewBorder(searchEntry, nil, nil, nil, accordion))
			newWindow.Canvas().Focus(searchEntry)
			newWindow.Show()
		}

		searchEntry.OnChanged("")
	}), msgEntry), nil, nil, messageList)
	return container.NewTabItem(channel, content)
}

func main() {
	a := app.New()
	w := a.NewWindow("hasherino2")
	w.Resize(fyne.NewSize(600, 800))
	w.SetMaster()

	hc := &hasherino.HasherinoController{}
	hc, err := hc.New(callbackMap)
	if err != nil {
		panic(err)
	}

	chatTabs := container.NewAppTabs()
	chatTabs.OnSelected = func(tab *container.TabItem) {
		hc.SetSelectedTab(tab.Text)
		chatTabs.Refresh()
	}

	sendMessage := func(message string) (string, error) {
		currentTab, err := hc.GetSelectedTab()
		if err != nil {
			return "", err
		}
		if !hc.IsChannelJoined(currentTab.Login) {
			return "", errors.New("Channel not joined. Please make sure you have an active account on settings.")
		}
		ac, err := hc.GetActiveAccount()
		if err != nil {
			return "", err
		}
		return ac.Login, hc.SendMessage(currentTab.Login, message)
	}

	savedTabs, err := hc.GetTabs()
	if err == nil {
		selectedTab, err := hc.GetSelectedTab()
		var tabIds []string

		for _, tab := range savedTabs {
			tabIds = append(tabIds, tab.Id)
			newTab := NewChatTab(tab.Login, sendMessage, hc.GetEmotes, w, hc.GetSettings)
			chatTabs.Append(newTab)
			if err == nil && selectedTab.Login == tab.Login {
				chatTabs.Select(newTab)
			}
		}
		tempTabErr := hc.AddTempTabs(&tabIds)
		if tempTabErr != nil {
			log.Println(tempTabErr)
		}

	}
	hc.Listen()

	components := container.NewBorder(
		container.NewHBox(
			widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), func() {
				dialog.ShowCustom("Settings", "Close", container.NewBorder(nil, nil, nil, nil, NewSettingsTabs(hc, w)), w)
			}),
			widget.NewButtonWithIcon("Add tab", theme.ContentAddIcon(), func() {
				entry := widget.NewEntry()
				items := []*widget.FormItem{
					widget.NewFormItem("New tab", entry),
				}
				var newTabDialog *dialog.FormDialog
				addTabFunc := func(b bool) {
					if b {
						err := hc.AddTab(entry.Text)
						if err != nil {
							dialog.ShowError(err, w)
						} else {
							chatTabs.Append(NewChatTab(entry.Text, sendMessage, hc.GetEmotes, w, hc.GetSettings))
							newTabDialog.Hide()
						}
					}
				}
				newTabDialog = dialog.NewForm("Add tab", "Add", "Cancel", items, addTabFunc, w)
				entry.SetPlaceHolder("Channel")
				entry.OnSubmitted = func(_ string) {
					addTabFunc(true)
				}
				newTabDialog.Show()

				w.Canvas().Focus(entry)
			}),
			widget.NewButtonWithIcon("Close tab", theme.CancelIcon(), func() {
				tab, err := hc.GetSelectedTab()
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				err = hc.RemoveTab(tab.Id)
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				delete(callbackMap, tab.Login)
				chatTabs.Remove(chatTabs.Selected())
			}),
		),

		nil,
		nil,
		nil,
		chatTabs,
	)

	w.SetContent(components)
	w.ShowAndRun()
}
