package main

import (
	"errors"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"fyne.io/fyne/v2/dialog"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/Hashy-Software/hasherino-go/hasherino"
)

var callbackMap = make(map[string]func(hasherino.ChatMessage))

func NewSettingsTabs(hc *hasherino.HasherinoController, w fyne.Window) *container.AppTabs {
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

	box := container.NewBorder(
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
	box.Add(table)

	tabs := container.NewAppTabs(
		container.NewTabItem("Accounts", box),
	)
	return tabs
}

func NewChatTab(name string, sendMsg func(string) (string, error), w fyne.Window) *container.TabItem {
	var data []string = []string{}
	messageList := widget.NewList(
		func() int {
			return len(data)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(data[i])
		})
	callbackMap[name] = func(message hasherino.ChatMessage) {
		if message.Command != "PRIVMSG" {
			return
		}
		text := message.Author + ": " + message.Text
		data = append(data, text)
		messageList.ScrollToBottom()
		messageList.Refresh()
	}
	msgEntry := widget.NewEntry()
	msgEntry.SetPlaceHolder("Message")
	msgEntry.Validator = func(s string) error {
		if len(s) > 500 {
			return errors.New("Message too long")
		}
		return nil
	}
	sendButton := widget.NewButton("Send", func() {
		text := msgEntry.Text

		err := msgEntry.Validate()
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		author, err := sendMsg(text)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		msgEntry.SetText("")
		data = append(data, author+": "+text)
		messageList.ScrollToBottom()
		messageList.Refresh()
	})
	msgEntry.OnSubmitted = func(_ string) {
		sendButton.OnTapped()
	}
	content := container.NewBorder(nil, container.NewBorder(nil, nil, nil, sendButton, msgEntry), nil, nil, messageList)
	return container.NewTabItem(name, content)
}

func main() {
	a := app.New()
	w := a.NewWindow("hasherino2")
	w.Resize(fyne.NewSize(400, 600))

	hc := &hasherino.HasherinoController{}
	hc, err := hc.New()
	if err != nil {
		panic(err)
	}

	settingsTab := container.NewTabItem("Settings", NewSettingsTabs(hc, w))
	tabs := container.NewAppTabs(
		settingsTab,
	)
	tabs.OnSelected = func(tab *container.TabItem) {
		if tab != settingsTab {
			hc.SetSelectedTab(tab.Text)
			tabs.Refresh()
		}
	}

	sendMessage := func(message string) (string, error) {
		currentTab, err := hc.GetSelectedTab()
		if err != nil {
			return "", err
		}
		if !hc.IsChannelJoined(currentTab.Login) {
			return "", errors.New("Channel not joined")
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
			newTab := NewChatTab(tab.Login, sendMessage, w)
			tabs.Append(newTab)
			if err == nil && selectedTab.Login == tab.Login {
				tabs.Select(newTab)
			}
		}
	}
	hc.Listen(callbackMap)

	// message list and buttons container
	components := container.NewBorder(
		widget.NewButton("Add tab", func() {
			entry := widget.NewEntry()
			entry.SetPlaceHolder("Channel")
			items := []*widget.FormItem{
				widget.NewFormItem("New tab", entry),
			}
			f := func(b bool) {
				if b {
					err := hc.AddTab(entry.Text)
					if err != nil {
						dialog.ShowError(err, w)
					} else {
						tabs.Append(NewChatTab(entry.Text, sendMessage, w))
					}
				}
			}
			dialog.ShowForm("Add tab", "Add", "Cancel", items, f, w)
		}),
		nil,
		nil,
		nil,
		tabs,
	)

	w.SetContent(components)
	w.ShowAndRun()

}
