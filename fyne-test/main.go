package main

import (
	"log"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	xWidget "fyne.io/x/fyne/widget"
)

type CustomWidget struct {
	widget.BaseWidget
	visible bool
	gif     *xWidget.AnimatedGif
	content *fyne.Container
}

func NewCustomWidget(text string) *CustomWidget {
	uri, err := storage.ParseURI("https://cdn.7tv.app/emote/61e66081095be332e347e5a4/4x.gif")

	if err != nil {
		panic(err)
	}
	widget, err := xWidget.NewAnimatedGif(uri)
	if err != nil {
		panic(err)
	}
	con := container.NewWithoutLayout(widget)
	widget.Resize(fyne.NewSize(50, 50))
	con.Resize(fyne.NewSize(50, 50))
	c := &CustomWidget{
		content: con,
		gif:     widget,
	}
	c.ExtendBaseWidget(c)
	widget.Start()
	return c
}

func (c *CustomWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.content)
}

func (c *CustomWidget) Refresh() {
	c.BaseWidget.Refresh()
	// Additional refresh logic if needed
}

func (c *CustomWidget) UpdateVisibility(scrollOffset fyne.Position, scrollSize fyne.Size, containerOffset fyne.Position) {
	widgetPos := c.Position()
	widgetSize := c.Size()

	isVisible := widgetPos.Y+widgetSize.Height > scrollOffset.Y &&
		widgetPos.Y < scrollOffset.Y+scrollSize.Height

	log.Println("visible:", isVisible)
	if isVisible {
		c.gif.Start()
	} else {
		c.gif.Stop()
	}
	c.visible = isVisible
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Visibility Example")

	customWidgets := []*CustomWidget{}
	for i := 0; i < 50; i++ {
		customWidgets = append(customWidgets, NewCustomWidget("Widget "+strconv.Itoa(i)))
	}

	vbox := container.NewVBox()
	for _, w := range customWidgets {
		vbox.Add(w)
		vbox.Add(widget.NewLabel(" "))
	}
	scrollContainer := container.NewVScroll(vbox)
	myWindow.SetContent(container.NewHSplit(scrollContainer, widget.NewButton("Visible?", func() {
		nVis := 0
		for _, w := range customWidgets {
			if w.visible {
				nVis++
			}
		}
		print(nVis)
	})))

	scrollContainer.OnScrolled = func(offset fyne.Position) {
		for _, w := range customWidgets {
			w.UpdateVisibility(offset, scrollContainer.Size(), scrollContainer.Offset)
		}
	}
	myWindow.Resize(fyne.NewSize(100, 100))
	myWindow.ShowAndRun()
	scrollContainer.OnScrolled(fyne.NewPos(0, 0))
}
