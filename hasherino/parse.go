package hasherino

import (
	"strings"

	"gopkg.in/irc.v4"
)

type ChatMessage struct {
	Channel string
	Command string
	Author  string
	Text    string
}

func ParseMessage(message string) (*ChatMessage, error) {
	msg, err := irc.ParseMessage(message)
	if err != nil {
		return nil, err
	}
	channel := ""
	if len(msg.Params) > 0 {
		if len(msg.Params[0]) > 0 && msg.Params[0][0] == '#' {
			channel = msg.Params[0][1:]
		} else {
			channel = msg.Params[0]
		}
	}
	paramsText := ""
	if len(msg.Params) > 1 {
		paramsText = strings.Join(msg.Params[1:], " ")
	}
	return &ChatMessage{
		Channel: channel,
		Command: msg.Command,
		Author:  msg.Name,
		Text:    paramsText,
	}, nil

}
