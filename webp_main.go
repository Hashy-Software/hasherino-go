package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/storage"
	"github.com/Hashy-Software/hasherino-go/components"
)

func main() {
	app := app.New()
	w := app.NewWindow("Animated GIF")
	uri, err := storage.ParseURI("https://cdn.7tv.app/emote/60a1babb3c3362f9a4b8b33a/4x.gif")
	if err != nil {
		log.Fatal(err)
	}
	g, err := components.NewWebpWidget(uri)
	if err != nil {
		log.Fatal(err)
	}
	g.Start()
	w.SetContent(g)
	w.Resize(fyne.NewSize(300, 300))
	w.ShowAndRun()
}
