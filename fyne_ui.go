package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	// "fyne.io/fyne/v2/dialog"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/Hashy-Software/hasherino-go/hasherino"

	"os"
	"time"
)

var data = []string{"a"}

// var list_data = [][]string{
// 	[]string{"top left", "top right"},
// 	[]string{"bottom left", "bottom right"},
// }

// func NewSettingsTabs(accounts []*hasherino.Account) *widget.FormItem {
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
	// item := widget.NewFormItem("", tabs)
	// Set item minsize
	// f := widget.NewForm(item)
	// f.Resize(fyne.NewSize(200, 300))
	return tabs
}

func main() {
	a := app.New()
	w := a.NewWindow("hasherino2")
	w.Resize(fyne.NewSize(400, 600))

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

	twitchChatWebsocket := &hasherino.TwitchChatWebsocket{}
	twitchChatWebsocket, err := twitchChatWebsocket.New(os.Getenv("TOKEN"), "hash_table")
	if err != nil {
		panic(err)
	}
	err = twitchChatWebsocket.Connect()
	if err != nil {
		panic(err)
	}
	i := time.Now()

	go func() {
		err = twitchChatWebsocket.Listen(func(message string) {
			println(message)
			data = append(data, message)
			if i.Add(time.Second / 2).Before(time.Now()) {
				i = time.Now()
				messageList.ScrollToBottom()
				messageList.Refresh()
			}
		})
		if err != nil {
			panic(err)
		}
	}()

	// message list and buttons container
	// components := container.NewBorder(
	// 	nil,
	// 	widget.NewButton("Join", func() {
	// 		twitchChatWebsocket.Join("hash_table")
	// 	}),
	// 	nil,
	// 	nil,
	// 	messageList,
	// )
	hc := &hasherino.HasherinoController{}
	hc, err = hc.New()

	tabs := container.NewAppTabs(
		container.NewTabItem("Settings", NewSettingsTabs(hc)),
	)
	w.SetContent(tabs)
	w.ShowAndRun()

}
