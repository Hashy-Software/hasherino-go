package components

import (
	"github.com/Hashy-Software/hasherino-go/hasherino"
)

func NewEmote(emote *hasherino.Emote, clickCallback func(string) error, lazyLoad bool) (LazyLoadedWidget, error) {
	if emote.Animated {
		return NewEmoteGif(emote, clickCallback, lazyLoad)
	}
	return NewWebpWidget(emote)
}
