package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"fyne.io/fyne/v2/dialog"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/Hashy-Software/hasherino-go/hasherino"
)

var callbackMap = make(map[string]func(hasherino.ChatMessage))

func NewSettingsTabs(hc *hasherino.HasherinoController) *container.AppTabs {
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

	box := container.NewBorder(
		nil,
		container.NewHBox(
			widget.NewButton("Add", func() {
				hc.OpenOAuthPage()
			}),
			widget.NewButton("Remove", func() {
				// TODO
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

func NewChatTab(name string) *container.TabItem {
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
	content := container.NewBorder(nil, nil, nil, nil, messageList)
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

	tabs := container.NewAppTabs(
		container.NewTabItem("Settings", NewSettingsTabs(hc)),
	)
	savedTabs, err := hc.GetTabs()
	if err == nil {
		for _, tab := range savedTabs {
			tabs.Append(NewChatTab(tab.DisplayName))
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
						tabs.Append(NewChatTab(entry.Text))
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
