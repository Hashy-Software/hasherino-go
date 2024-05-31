package hasherino_test

import (
	"testing"

	"github.com/Hashy-Software/hasherino-go/hasherino"
)

func TestGetEmojiJson(t *testing.T) {
	emojiJSON, err := hasherino.GetEmojiJSONMap()
	if err != nil {
		t.Error(err)
	}
	if emojiJSON == nil {
		t.Error("emojiJson is nill")
	}
}
