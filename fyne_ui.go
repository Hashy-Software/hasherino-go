package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/Hashy-Software/hasherino-go/hasherino"

	"os"
)

var data = []string{"a"}

func main() {
	a := app.New()
	w := a.NewWindow("hasherino2")

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

	go func() {
		err = twitchChatWebsocket.Listen(func(message string) {
			println(message)
			data = append(data, message)
			messageList.Refresh()
		})
		if err != nil {
			panic(err)
		}
	}()

	// message list and buttons container
	components := container.NewBorder(
		nil,
		widget.NewButton("Join", func() {
			twitchChatWebsocket.Join("hash_table")
		}),
		nil,
		nil,
		messageList,
	)

	tabs := container.NewAppTabs(
		container.NewTabItem("hash_table", components),
	)
	w.SetContent(tabs)
	w.ShowAndRun()

}
