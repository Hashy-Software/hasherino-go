package hasherino

import (
	"errors"

	"context"
	"fmt"
	"io"
	"nhooyr.io/websocket"
)

type WebsocketState int64

const (
	Disconnected WebsocketState = iota
	Connected
)

type TwitchChatWebsocket struct {
	State WebsocketState

	url              string
	context          context.Context
	cancel           context.CancelFunc
	connection       *websocket.Conn
	initial_messages *[]string
	channels         map[string]struct{}
}

func (w *TwitchChatWebsocket) New(token string, user string) (*TwitchChatWebsocket, error) {
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
	return &TwitchChatWebsocket{
		url:              url,
		State:            Disconnected,
		context:          ctx,
		cancel:           cancel,
		connection:       c,
		initial_messages: &initial_messages,
		channels:         make(map[string]struct{}),
	}, nil
}

func (w *TwitchChatWebsocket) Connect() error {
	if w.State != Disconnected {
		return errors.New("Not disconnected")
	}
	for _, msg := range *w.initial_messages {
		err := w.connection.Write(w.context, websocket.MessageText, []byte(msg))
		if err != nil {
			return err
		}
	}
	w.State = Connected
	return nil
}

func (w *TwitchChatWebsocket) Close() {
	w.cancel()
	w.connection.Close(websocket.StatusNormalClosure, "")
	w.State = Disconnected
}

func (w *TwitchChatWebsocket) Listen(callback func(message string)) error {
	if w.State != Connected {
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
		fmt.Println("Message: " + s)
		callback(s)
	}
	w.cancel()
	w.connection.Close(websocket.StatusNormalClosure, "")

	return nil
}

func (w *TwitchChatWebsocket) Join(channel string) error {
	if w.State != Connected {
		return errors.New("Not connected")
	}
	err := w.connection.Write(w.context, websocket.MessageText, []byte("JOIN #"+channel))
	if err != nil {
		return err
	}
	w.channels[channel] = struct{}{}
	return nil

}

func (w *TwitchChatWebsocket) Part(channel string) error {
	if w.State != Connected {
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

func (w *TwitchChatWebsocket) Send(channel string, message string) error {
	if w.State != Connected {
		return errors.New("Not connected")
	}
	err := w.connection.Write(w.context, websocket.MessageText, []byte("PRIVMSG #"+channel+" :"+message))
	if err != nil {
		return err
	}
	return nil
}
