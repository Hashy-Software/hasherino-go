package hasherino

import (
	"errors"

	"gorm.io/gorm"
)

// --- permDB models ---
type Account struct {
	Id          string `gorm:"primaryKey"`
	Login       string
	DisplayName string
	Active      bool
	Token       string // TODO: enable db encryption to hide token
}

type Tab struct {
	Id          string `gorm:"primaryKey"`
	Login       string
	DisplayName string
	Selected    bool
}

// Single row table for global settings
type AppSettings struct {
	gorm.Model
	ChatMessageLimit int // Maximum amount of messages in a single chat
	ChatHistory      bool
}

// --- tempDB models ---
type EmoteSourceEnum int64

const (
	Twitch EmoteSourceEnum = iota
	SevenTV
)

type ChatUser struct {
	Id          string `gorm:"primaryKey"`
	Login       string
	DisplayName string
	Emotes      []Emote `gorm:"constraint:OnDelete:CASCADE,OnUpdate:CASCADE"`
}

type Emote struct {
	Id        string          `gorm:"primaryKey"`
	Source    EmoteSourceEnum `gorm:"primaryKey"`
	Name      string
	ChannelID *string   // if a channel is set, only renders in that channel
	Owner     *ChatUser // if an owner is set, only renders when the message sender is the owner
}

func (e *Emote) GetUrl() (string, error) {
	switch e.Source {
	case Twitch:
		return "https://static-cdn.jtvnw.net/emoticons/v2/" + e.Id + "/default/dark/2.0", nil
	case SevenTV:
		return "https://cdn.7tv.app/emote/" + e.Id + "/2x.webp", nil
	default:
		return "", errors.New("Unknown emote source")
	}
}
