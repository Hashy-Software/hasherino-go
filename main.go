package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	// "fyne.io/fyne/v2/storage"

	"fyne.io/fyne/v2/container"
	// "fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	// xwidget "fyne.io/x/fyne/widget"
	"context"
	"fmt"
	"io"
	"nhooyr.io/websocket"
	"os"
	"time"
)

var data = []string{"a"}

func connect(list *widget.List) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	c, _, err := websocket.Dial(ctx, "wss://irc-ws.chat.twitch.tv", nil)
	if err != nil {
		panic(err)
	}
	defer c.CloseNow()

	token := os.Getenv("TOKEN")
	if token == "" {
		panic("Token envvar not set")
	}

	initial_messages := []string{
		"CAP REQ :twitch.tv/commands twitch.tv/tags",
		"PASS oauth:" + token,
		"NICK hash_table",
		"JOIN #hash_table",
	}

	for _, msg := range initial_messages {
		err = c.Write(ctx, websocket.MessageText, []byte(msg))
		if err != nil {
			panic(err)
		}
	}

	defer c.Close(websocket.StatusNormalClosure, "")

	for {
		_, content, err := c.Read(ctx)
		if err != nil && err != io.EOF {
			fmt.Println("Error:", err)
			break
		}
		if err == io.EOF {
			fmt.Println("EOF, continuing")
			continue
		}
		data = append(data, string(content))
		fmt.Println("Message: " + string(content))
		list.Refresh()
	}

}

func main() {
	a := app.New()
	w := a.NewWindow("Hello World")

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

	// message list and buttons container
	components := container.NewVBox()
	components.Add(widget.NewButton("Connect", func() { connect(messageList) }))
	components.Add(messageList)

	tabs := container.NewAppTabs(
		container.NewTabItem("hash_table", components),
	)
	w.SetContent(tabs)
	w.ShowAndRun()

}
