package components

import (
	"bytes"
	"image"
	"image/draw"
	"image/gif"
	"io"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// WebpWidget widget shows a Gif image with many frames.
type WebpWidget struct {
	widget.BaseWidget
	min fyne.Size

	src               *gif.GIF
	dst               *canvas.Image
	noDisposeIndex    int
	remaining         int
	stopping, running bool
	runLock           sync.RWMutex
}

// NewWebpWidget creates a new widget loaded to show the specified image.
// If there is an error loading the image it will be returned in the error value.
func NewWebpWidget(u fyne.URI) (*WebpWidget, error) {
	ret := newAnimatedGif()

	return ret, ret.Load(u)
}

// NewWebpWidgetFromResource creates a new widget loaded to show the specified image resource.
// If there is an error loading the image it will be returned in the error value.
func NewWebpWidgetFromResource(r fyne.Resource) (*WebpWidget, error) {
	ret := newAnimatedGif()

	return ret, ret.LoadResource(r)
}

// CreateRenderer loads the widget renderer for this widget. This is an internal requirement for Fyne.
func (g *WebpWidget) CreateRenderer() fyne.WidgetRenderer {
	return &WebpRenderer{webp: g}
}

// Load is used to change the gif file shown.
// It will change the loaded content and prepare the new frames for animation.
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

// LoadResource is used to change the gif resource shown.
// It will change the loaded content and prepare the new frames for animation.
func (g *WebpWidget) LoadResource(r fyne.Resource) error {
	g.dst.Image = nil
	g.dst.Refresh()

	if r == nil || len(r.Content()) == 0 {
		return nil
	}
	return g.load(bytes.NewReader(r.Content()))
}

func (g *WebpWidget) load(read io.Reader) error {
	pix, err := gif.DecodeAll(read)
	if err != nil {
		return err
	}
	g.src = pix
	g.dst.Image = pix.Image[0]
	g.dst.Refresh()

	return nil
}

// MinSize returns the minimum size that this GIF can occupy.
// Because gif images are measured in pixels we cannot use the dimensions, so this defaults to 0x0.
// You can set a minimum size if required using SetMinSize.
func (g *WebpWidget) MinSize() fyne.Size {
	return g.min
}

// SetMinSize sets the smallest possible size that this AnimatedGif should be drawn at.
// Be careful not to set this based on pixel sizes as that will vary based on output device.
func (g *WebpWidget) SetMinSize(min fyne.Size) {
	g.min = min
}

func (g *WebpWidget) draw(dst draw.Image, index int) {
	defer g.dst.Refresh()
	if index == 0 {
		// first frame
		draw.Draw(dst, g.dst.Image.Bounds(), g.src.Image[index], image.Point{}, draw.Src)
		g.dst.Image = dst
		g.noDisposeIndex = -1
		return
	}

	switch g.src.Disposal[index-1] {
	case gif.DisposalNone:
		// Do not dispose old frame, draw new frame over old
		draw.Draw(dst, g.dst.Image.Bounds(), g.src.Image[index], image.Point{}, draw.Over)
		// will be used in case of disposalPrevious
		g.noDisposeIndex = index - 1
	case gif.DisposalBackground:
		// clear with background then render new frame Over it
		// replacing entirely with new frame should achieve this?
		draw.Draw(dst, g.dst.Image.Bounds(), g.src.Image[index], image.Point{}, draw.Src)
	case gif.DisposalPrevious:
		// restore frame with previous image then render new over it
		if g.noDisposeIndex >= 0 {
			draw.Draw(dst, g.dst.Image.Bounds(), g.src.Image[g.noDisposeIndex], image.Point{}, draw.Src)
			draw.Draw(dst, g.dst.Image.Bounds(), g.src.Image[index], image.Point{}, draw.Over)
		} else {
			// there was no previous graphic, render background instead?
			draw.Draw(dst, g.dst.Image.Bounds(), g.src.Image[index], image.Point{}, draw.Src)
		}
	default:
		// Disposal = Unspecified/Reserved, simply draw new frame over previous
		draw.Draw(dst, g.dst.Image.Bounds(), g.src.Image[index], image.Point{}, draw.Over)
	}
}

// Start begins the animation. The speed of the transition is controlled by the loaded gif file.
func (g *WebpWidget) Start() {
	if g.isRunning() {
		return
	}
	g.runLock.Lock()
	g.running = true
	g.runLock.Unlock()

	buffer := image.NewNRGBA(g.dst.Image.Bounds())
	g.draw(buffer, 0)

	go func() {
		switch g.src.LoopCount {
		case -1: // don't loop
			g.remaining = 1
		case 0: // loop forever
			g.remaining = -1
		default:
			g.remaining = g.src.LoopCount + 1
		}
	loop:
		for g.remaining != 0 {
			for c := range g.src.Image {
				if g.isStopping() {
					break loop
				}
				g.draw(buffer, c)

				time.Sleep(time.Millisecond * time.Duration(g.src.Delay[c]) * 10)
			}
			if g.remaining > -1 { // don't underflow int
				g.remaining--
			}
		}
		g.runLock.Lock()
		g.running = false
		g.stopping = false
		g.runLock.Unlock()
	}()
}

// Stop will request that the animation stops running, the last frame will remain visible
func (g *WebpWidget) Stop() {
	if !g.isRunning() {
		return
	}
	g.runLock.Lock()
	g.stopping = true
	g.runLock.Unlock()
}

func (g *WebpWidget) isStopping() bool {
	g.runLock.RLock()
	defer g.runLock.RUnlock()
	return g.stopping
}

func (g *WebpWidget) isRunning() bool {
	g.runLock.RLock()
	defer g.runLock.RUnlock()
	return g.running
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
	g.webp.Stop()
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
