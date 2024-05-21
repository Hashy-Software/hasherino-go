package main

import (
	"bytes"
	"image"
	"image/draw"
	"image/gif"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// AnimatedGif widget shows a Gif image with many frames.
type AnimatedGif struct {
	widget.BaseWidget
	min fyne.Size

	src               *gif.GIF
	dst               *canvas.Image
	noDisposeIndex    int
	remaining         int
	stopping, running bool
	runLock           sync.RWMutex
}

// NewAnimatedGif creates a new widget loaded to show the specified image.
// If there is an error loading the image it will be returned in the error value.
func NewAnimatedGif(u fyne.URI) (*AnimatedGif, error) {
	ret := newGif()

	return ret, ret.Load(u)
}

// NewAnimatedGifFromResource creates a new widget loaded to show the specified image resource.
// If there is an error loading the image it will be returned in the error value.
func NewAnimatedGifFromResource(r fyne.Resource) (*AnimatedGif, error) {
	ret := newGif()

	return ret, ret.LoadResource(r)
}

// CreateRenderer loads the widget renderer for this widget. This is an internal requirement for Fyne.
func (g *AnimatedGif) CreateRenderer() fyne.WidgetRenderer {
	return &gifRenderer{gif: g}
}

// Load is used to change the gif file shown.
// It will change the loaded content and prepare the new frames for animation.
func (g *AnimatedGif) Load(u fyne.URI) error {
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
func (g *AnimatedGif) LoadResource(r fyne.Resource) error {
	g.dst.Image = nil
	g.dst.Refresh()

	if r == nil || len(r.Content()) == 0 {
		return nil
	}
	return g.load(bytes.NewReader(r.Content()))
}

func (g *AnimatedGif) load(read io.Reader) error {
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
func (g *AnimatedGif) MinSize() fyne.Size {
	return g.min
}

// SetMinSize sets the smallest possible size that this AnimatedGif should be drawn at.
// Be careful not to set this based on pixel sizes as that will vary based on output device.
func (g *AnimatedGif) SetMinSize(min fyne.Size) {
	g.min = min
}

func (g *AnimatedGif) draw(dst draw.Image, index int) {
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
func (g *AnimatedGif) Start() {
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
func (g *AnimatedGif) Stop() {
	if !g.isRunning() {
		return
	}
	g.runLock.Lock()
	g.stopping = true
	g.runLock.Unlock()
}

func (g *AnimatedGif) isStopping() bool {
	g.runLock.RLock()
	defer g.runLock.RUnlock()
	return g.stopping
}

func (g *AnimatedGif) isRunning() bool {
	g.runLock.RLock()
	defer g.runLock.RUnlock()
	return g.running
}

func newGif() *AnimatedGif {
	ret := &AnimatedGif{}
	ret.ExtendBaseWidget(ret)
	ret.dst = &canvas.Image{}
	ret.dst.FillMode = canvas.ImageFillContain
	return ret
}

type gifRenderer struct {
	gif *AnimatedGif
}

func (g *gifRenderer) Destroy() {
	g.gif.Stop()
}

func (g *gifRenderer) Layout(size fyne.Size) {
	g.gif.dst.Resize(size)
}

func (g *gifRenderer) MinSize() fyne.Size {
	return g.gif.MinSize()
}

func (g *gifRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{g.gif.dst}
}

func (g *gifRenderer) Refresh() {
	g.gif.dst.Refresh()
}

var resourceCache = make(map[string]string)

func ResourceLoader(gifUrl string) (*fyne.StaticResource, error) {
	var (
		uri fyne.URI
		err error
	)

	if res, ok := resourceCache[gifUrl]; ok {
		bytes, err := os.ReadFile(res)
		if err != nil {
			return nil, err
		}
		return fyne.NewStaticResource(gifUrl, bytes), nil
	} else {
		uri, err = storage.ParseURI(gifUrl)
		if err != nil {
			return nil, err
		}
		read, err := storage.Reader(uri)
		if err != nil {
			return nil, err
		}
		bytes, err := io.ReadAll(read)
		if err != nil {
			return nil, err
		}
		tempFile, err := os.CreateTemp("", "")
		if err != nil {
			return nil, err
		}
		err = os.WriteFile(tempFile.Name(), bytes, 0666)
		if err != nil {
			return nil, err
		}
		resourceCache[gifUrl] = tempFile.Name()
		log.Println("Caching", gifUrl, "to", tempFile.Name())
		return &fyne.StaticResource{StaticName: gifUrl, StaticContent: bytes}, nil
	}
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Visibility Example")

	/*
		TODO:
			- [X] Cache images to temp files
			- [ ] Use the same function to draw all gifs
			- [ ] Lazy load images
				- [ ] Return from New immediatly with a temporary blank image with specified size
				- [ ] On Start, load image
			- [ ] Try loading from webp for better size and performance
	*/
	var urls *[]string = &[]string{
		"https://cdn.7tv.app/emote/651f30352afe76536598abf0/4x.gif",
		"https://cdn.7tv.app/emote/651f30352afe76536598abf0/4x.gif",
		"https://cdn.7tv.app/emote/651f30352afe76536598abf0/4x.gif",
		"https://cdn.7tv.app/emote/651f30352afe76536598abf0/4x.gif",
		"https://cdn.7tv.app/emote/60a1babb3c3362f9a4b8b33a/4x.gif",
		"https://cdn.7tv.app/emote/60a1babb3c3362f9a4b8b33a/4x.gif",
		"https://cdn.7tv.app/emote/60a1babb3c3362f9a4b8b33a/4x.gif",
		"https://cdn.7tv.app/emote/60a1babb3c3362f9a4b8b33a/4x.gif",
	}
	imgContainer := container.NewVBox()

	for _, url := range *urls {
		res, err := ResourceLoader(url)
		if err != nil {
			panic(err)
		}
		g, err := NewAnimatedGifFromResource(res)
		if err != nil {
			panic(err)
		}
		// u, err := storage.ParseURI("https://cdn.7tv.app/emote/651f30352afe76536598abf0/2x.gif")
		// if err != nil {
		// 	panic(err)
		// }
		// g, err := NewAnimatedGif(u)
		// if err != nil {
		// 	panic(err)
		// }
		g.Start()
		c := container.NewWithoutLayout(g)
		c.Resize(fyne.NewSize(64, 64))
		g.Resize(fyne.NewSize(64, 64))
		imgContainer.Add(c)
		imgContainer.Add(widget.NewLabel(" "))
		imgContainer.Add(widget.NewLabel(" "))

	}

	myWindow.SetContent(container.NewAppTabs(
		container.NewTabItem("Images", imgContainer),
		container.NewTabItem("ASD", container.NewVBox()),
	))
	myWindow.ShowAndRun()
}
