package components

import (
	"bytes"
	"golang.org/x/image/webp"
	"io"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// WebpWidget widget shows a Gif image with many frames.
type WebpWidget struct {
	widget.BaseWidget
	min fyne.Size

	dst *canvas.Image
}

func NewWebpWidget(u fyne.URI) (*WebpWidget, error) {
	ret := newAnimatedGif()

	return ret, ret.Load(u)
}

func NewWebpWidgetFromResource(r fyne.Resource) (*WebpWidget, error) {
	ret := newAnimatedGif()

	return ret, ret.LoadResource(r)
}

func (g *WebpWidget) CreateRenderer() fyne.WidgetRenderer {
	return &WebpRenderer{webp: g}
}

func (g *WebpWidget) Load(u fyne.URI) error {
	g.dst.Image = nil
	g.dst.Refresh()

	if u == nil {
		return nil
	}

	read, err := storage.Reader(u)
	if err != nil {
		return err
	}

	return g.load(read)
}

func (g *WebpWidget) LoadResource(r fyne.Resource) error {
	g.dst.Image = nil
	g.dst.Refresh()

	if r == nil || len(r.Content()) == 0 {
		return nil
	}
	return g.load(bytes.NewReader(r.Content()))
}

func (g *WebpWidget) load(read io.Reader) error {
	pix, err := webp.Decode(read)
	if err != nil {
		return err
	}
	g.dst.Image = pix
	g.dst.Refresh()

	return nil
}

func (g *WebpWidget) MinSize() fyne.Size {
	return g.min
}

func (g *WebpWidget) SetMinSize(min fyne.Size) {
	g.min = min
}

func newAnimatedGif() *WebpWidget {
	ret := &WebpWidget{}
	ret.ExtendBaseWidget(ret)
	ret.dst = &canvas.Image{}
	ret.dst.FillMode = canvas.ImageFillContain
	return ret
}

type WebpRenderer struct {
	webp *WebpWidget
}

func (g *WebpRenderer) Destroy() {
}

func (g *WebpRenderer) Layout(size fyne.Size) {
	g.webp.dst.Resize(size)
}

func (g *WebpRenderer) MinSize() fyne.Size {
	return g.webp.MinSize()
}

func (g *WebpRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{g.webp.dst}
}

func (g *WebpRenderer) Refresh() {
	g.webp.dst.Refresh()
}
