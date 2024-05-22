package main

import (
	"log"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/Hashy-Software/hasherino-go/components"
	"github.com/Hashy-Software/hasherino-go/hasherino"
)

func main() {
	app := app.New()
	w := app.NewWindow("Animated GIF")
	t, err := os.CreateTemp("", "")
	if err != nil {
		log.Fatal(err)
	}
	emote := &hasherino.Emote{
		Id:       "663d3e7efcc4ab2ae6dc0428",
		Source:   hasherino.SevenTV,
		Animated: false,
		TempFile: t.Name(),
	}
	var l []components.LazyLoadedWidget
	for range 500 {
		g, err := components.NewWebpWidget(emote)
		if err != nil {
			log.Fatal(err)
		}
		g.LazyLoad()
		l = append(l, g)
		w.SetContent(g)
		g.LazyUnload()
	}
	w.Content().(*components.WebpWidget).LazyLoad()
	w.Resize(fyne.NewSize(300, 300))
	w.ShowAndRun()
}
