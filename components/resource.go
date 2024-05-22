package components

import (
	"io"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"github.com/Hashy-Software/hasherino-go/hasherino"
)

func tempFileResource(emote *hasherino.Emote) (*fyne.StaticResource, error) {
	s, err := os.Stat(emote.TempFile)
	if err != nil {
		return nil, err
	}
	url, err := emote.GetUrl()
	if err != nil {
		return nil, err
	}
	// image hasn't been written to tempfile yet
	if s.Size() == 0 {
		parsedUri, err := storage.ParseURI(url)
		if err != nil {
			return nil, err
		}
		read, err := storage.Reader(parsedUri)
		if err != nil {
			return nil, err
		}
		bytes, err := io.ReadAll(read)
		if err != nil {
			return nil, err
		}
		err = os.WriteFile(emote.TempFile, bytes, 0666)
		if err != nil {
			return nil, err
		}
		res := fyne.StaticResource{StaticName: url, StaticContent: bytes}
		return &res, nil
	}
	parsedUri, err := storage.ParseURI("file://" + emote.TempFile)
	if err != nil {
		return nil, err
	}
	read, err := storage.Reader(parsedUri)
	if err != nil {
		return nil, err
	}
	bytes, err := io.ReadAll(read)
	if err != nil {
		return nil, err
	}
	return &fyne.StaticResource{StaticName: url, StaticContent: bytes}, err
}
