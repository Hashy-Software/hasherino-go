package main

import (
	// "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/storage"

	"fyne.io/fyne/v2/container"
	// "fyne.io/fyne/v2/layout"
	// "fyne.io/fyne/v2/widget"
	xwidget "fyne.io/x/fyne/widget"
)

var data = []string{"a"}

func main() {
	a := app.New()
	w := a.NewWindow("Hello World")

	gif, err := xwidget.NewAnimatedGif(storage.NewFileURI("/home/douglas/Downloads/pats.gif"))
	if err == nil {
		gif.Start()
		// w.SetContent(gif)
	} else {
		panic(err)
	}

	// c := widget.NewList(
	// 	func() int {
	// 		return len(data)
	// 	},
	// 	func() fyne.CanvasObject {
	// 		a, _ := xwidget.NewAnimatedGif(storage.NewFileURI("/home/douglas/Downloads/pats.gif"))
	// 		a.Start()
	// 		a.Resize(fyne.NewSize(100, 100))
	// 		return a
	// 	},
	// 	func(i widget.ListItemID, o fyne.CanvasObject) {
	// 		// o.(*xwidget.AnimatedGif).Start()
	// 		// o.(*widget.Label).SetText(data[i])
	// 	},
	// )
	tabs := container.NewAppTabs(
		container.NewTabItem("hash_table", gif),
		// container.NewTabItem("hash_table", gif),
	)
	w.SetContent(tabs)

	w.ShowAndRun()
}
