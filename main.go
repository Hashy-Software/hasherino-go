package main

import (
	"errors"
	"strings"

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
)

var data = []string{"a"}

type WebsocketState int64

const (
	Disconnected WebsocketState = iota
	Connected
)

type HasherinoWebsocket struct {
	url              string
	state            WebsocketState
	context          context.Context
	cancel           context.CancelFunc
	connection       *websocket.Conn
	initial_messages *[]string
	channels         map[string]struct{}
}

func (w *HasherinoWebsocket) New(token string, user string) (*HasherinoWebsocket, error) {
	initial_messages := []string{
		"CAP REQ :twitch.tv/commands twitch.tv/tags",
		"PASS oauth:" + token,
		"NICK " + user,
	}
	url := "wss://irc-ws.chat.twitch.tv"
	ctx, cancel := context.WithCancel(context.Background())
	c, _, err := websocket.Dial(ctx, url, nil)

	if err != nil {
		return nil, err
	}
	return &HasherinoWebsocket{
		url:              url,
		state:            Disconnected,
		context:          ctx,
		cancel:           cancel,
		connection:       c,
		initial_messages: &initial_messages,
	}, nil
}

func (w *HasherinoWebsocket) Connect() error {
	if w.state != Disconnected {
		return errors.New("Not disconnected")
	}
	for _, msg := range *w.initial_messages {
		err := w.connection.Write(w.context, websocket.MessageText, []byte(msg))
		if err != nil {
			return err
		}
	}
	w.state = Connected
	return nil
}

func (w *HasherinoWebsocket) Listen(callback func(message string)) error {
	if w.state != Connected {
		return errors.New("Not connected")
	}

	for {
		_, content, err := w.connection.Read(w.context)
		if err != nil && err != io.EOF {
			fmt.Println("Error:", err)
			break
		}
		if err == io.EOF {
			fmt.Println("EOF, continuing")
			continue
		}
		s := string(content)
		privmsgIndex := strings.Index(s, "PRIVMSG")
		if privmsgIndex > -1 {
			data = append(data, s[privmsgIndex+len("PRIVMSG"):])
		}
		fmt.Println("Message: " + s)
		callback(s)
	}
	w.cancel()
	w.connection.Close(websocket.StatusNormalClosure, "")

	return nil
}

func (w *HasherinoWebsocket) Join(channel string) error {
	if w.state != Connected {
		return errors.New("Not connected")
	}
	err := w.connection.Write(w.context, websocket.MessageText, []byte("JOIN #"+channel))
	if err != nil {
		return err
	}
	return nil

}

func (w *HasherinoWebsocket) Part(channel string) error {
	if w.state != Connected {
		return errors.New("Not connected")
	}
	_, ok := w.channels[channel]
	if !ok {
		return errors.New("Not in channel")
	}

	err := w.connection.Write(w.context, websocket.MessageText, []byte("PART #"+channel))
	if err != nil {
		return err
	}

	delete(w.channels, channel)
	return nil

}

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

	twitchWebsocket := &HasherinoWebsocket{}
	twitchWebsocket, err := twitchWebsocket.New(os.Getenv("TOKEN"), "hash_table")
	if err != nil {
		panic(err)
	}
	err = twitchWebsocket.Connect()
	if err != nil {
		panic(err)
	}

	go func() {
		err = twitchWebsocket.Listen(func(message string) {
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
			twitchWebsocket.Join("hash_table")
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
