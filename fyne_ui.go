package main

import (
	"errors"
	"log"
	"net/url"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"

	"fyne.io/fyne/v2/dialog"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/Hashy-Software/hasherino-go/hasherino"
)

var callbackMap = make(map[string]func(hasherino.ChatMessage))

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

func NewChatTab(
	channel string,
	sendMsg func(string) (string, error),
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
	content := container.NewBorder(nil, container.NewBorder(nil, nil, nil, nil, msgEntry), nil, nil, messageList)
	return container.NewTabItem(channel, content)
}

func main() {
	a := app.New()
	w := a.NewWindow("hasherino2")
	w.Resize(fyne.NewSize(600, 800))

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
		for _, tab := range savedTabs {
			newTab := NewChatTab(tab.Login, sendMessage, w, hc.GetSettings)
			chatTabs.Append(newTab)
			if err == nil && selectedTab.Login == tab.Login {
				chatTabs.Select(newTab)
			}
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
							chatTabs.Append(NewChatTab(entry.Text, sendMessage, w, hc.GetSettings))
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
