package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"log"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten"
	"github.com/martinkirsche/wired-logic/simulation"
)

var (
	simulationTimer        <-chan time.Time
	simulationImage        *image.Paletted
	currentSimulation      *simulation.Simulation
	backgroundImage        *ebiten.Image
	wireImages             []*ebiten.Image
	wasMouseButtonPressed  = false
	cursorBlinking         uint8
	cursorImage            *ebiten.Image
	oldMouseCursorPosition image.Point = image.Point{-1, -1}
	cursorPosition         image.Point = image.Point{-1, -1}
	keyStates                          = map[ebiten.Key]int{
		ebiten.KeyUp:    0,
		ebiten.KeyDown:  0,
		ebiten.KeyLeft:  0,
		ebiten.KeyRight: 0,
		ebiten.KeySpace: 0,
		ebiten.KeyW:     0,
		ebiten.KeyA:     0,
		ebiten.KeyS:     0,
		ebiten.KeyD:     0,
		ebiten.KeyP:     0,
	}
	simulationPaused = false
)

func main() {
	var err error
	if cursorImage, err = ebiten.NewImage(4, 4, ebiten.FilterNearest); err != nil {
		log.Fatal(err)
	}
	cursorImage.Fill(color.White)
	var speed, scale, width, height int
	flag.IntVar(&speed, "speed", 15, "simulation steps per second")
	flag.IntVar(&scale, "scale", 16, "pixel scale factor")
	flag.IntVar(&width, "width", 64, "width of the simulation")
	flag.IntVar(&height, "height", 64, "height of the simulation")
	flag.Parse()
	flag.Args()

	cursorPosition = image.Point{width / 2, height / 2}
	simulationTimer = time.Tick(time.Second / time.Duration(speed))
	if flag.NArg() == 1 {
		inputFileName := flag.Arg(0)
		in, err := os.Open(inputFileName)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		gifImage, err := gif.DecodeAll(in)
		if err != nil {
			log.Fatal(err)
		}
		simulationImage = gifImage.Image[0]
		simulationImage.Palette[0] = color.Transparent
	} else {
		p := color.Palette{
			color.Black,
			color.RGBA{0x88, 0, 0, 0xFF},
			color.RGBA{0xFF, 0, 0, 0xFF},
			color.RGBA{0xFF, 0x22, 0, 0xFF},
			color.RGBA{0xFF, 0x44, 0,  0xFF},
			color.RGBA{0xFF, 0x66, 0,  0xFF},
			color.RGBA{0xFF, 0x88, 0, 0xFF},
			color.RGBA{0xFF, 0xAA, 0,  0xFF},
		}
		simulationImage = image.NewPaletted(image.Rect(0, 0, width, height), p)
	}
	reloadSimulation()
	if err := ebiten.Run(update, simulationImage.Bounds().Dx(), simulationImage.Bounds().Dy(), float64(scale), "Wired Logic"); err != nil {
		log.Fatal(err)
	}
}

func reloadSimulation() error {
	currentSimulation = simulation.New(simulationImage)
	currentSimulation.Draw(simulationImage)
	var err error
	backgroundImage, err = ebiten.NewImageFromImage(simulationImage, ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	for _, img := range wireImages {
		if err = img.Dispose(); err != nil {
			return err
		}
	}
	wires := currentSimulation.Circuit().Wires()
	wireImages = make([]*ebiten.Image, len(wires))
	for i, wire := range wires {
		img := drawMask(wire)
		var err error
		if wireImages[i], err = ebiten.NewImageFromImage(img, ebiten.FilterNearest); err != nil {
			return err
		}
	}
	return nil
}

func togglePixel(position image.Point) error {
	currentSimulation.Draw(simulationImage)
	c := simulationImage.ColorIndexAt(position.X, position.Y)
	if c-1 > simulation.MaxCharge {
		simulationImage.SetColorIndex(position.X, position.Y, 1)
	} else {
		simulationImage.SetColorIndex(position.X, position.Y, 0)
	}
	if err := reloadSimulation(); err != nil {
		return err
	}
	return nil
}

func readKeys() {
	for key, _ := range keyStates {
		if !ebiten.IsKeyPressed(key) {
			keyStates[key] = -1
			continue
		}
		keyStates[key]++

	}
}

func handleCursor(screen *ebiten.Image) error {
	mx, my := ebiten.CursorPosition()
	cursorMoved := image.Point{mx, my}.In(screen.Bounds()) && (mx != oldMouseCursorPosition.X || my != oldMouseCursorPosition.Y)
	oldMouseCursorPosition = image.Point{mx, my}
	if cursorMoved {
		cursorPosition = oldMouseCursorPosition
	} else {
		const cursorInterval = 6
		switch {
		case keyStates[ebiten.KeyUp]%cursorInterval == 0 ||
			keyStates[ebiten.KeyW]%cursorInterval == 0:
			cursorPosition = cursorPosition.Add(image.Point{0, -1})
			cursorMoved = true
		case keyStates[ebiten.KeyDown]%cursorInterval == 0 ||
			keyStates[ebiten.KeyS]%cursorInterval == 0:
			cursorPosition = cursorPosition.Add(image.Point{0, +1})
			cursorMoved = true
		case keyStates[ebiten.KeyLeft]%cursorInterval == 0 ||
			keyStates[ebiten.KeyA]%cursorInterval == 0:
			cursorPosition = cursorPosition.Add(image.Point{-1, 0})
			cursorMoved = true
		case keyStates[ebiten.KeyRight]%cursorInterval == 0 ||
			keyStates[ebiten.KeyD]%cursorInterval == 0:
			cursorPosition = cursorPosition.Add(image.Point{+1, 0})
			cursorMoved = true
		}
	}
	if cursorBlinking == 127 {
		cursorBlinking = 0
	} else {
		cursorBlinking++
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(0.25, .25)
	op.GeoM.Translate(float64(cursorPosition.X), float64(cursorPosition.Y))
	if cursorBlinking > 64 {
		op.ColorM.Scale(1, 1, 1, 0.25+float64(127-cursorBlinking)/255.0)
	} else {
		op.ColorM.Scale(1, 1, 1, 0.25+float64(cursorBlinking)/255.0)
	}
	if err := screen.DrawImage(cursorImage, op); err != nil {
		return err
	}
	if keyStates[ebiten.KeySpace] >= 0 ||
		ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if cursorMoved || !wasMouseButtonPressed {
			if err := togglePixel(cursorPosition); err != nil {
				return err
			}
			wasMouseButtonPressed = true
		}
	} else {
		wasMouseButtonPressed = false
	}
	return nil
}

func update(screen *ebiten.Image) error {
	readKeys()
	if keyStates[ebiten.KeyP] == 0 {
		simulationPaused = !simulationPaused
	}
	select {
	case <-simulationTimer:
		newSimulation := currentSimulation
		if !simulationPaused {
			newSimulation = currentSimulation.Step()
		}
		wires := currentSimulation.Circuit().Wires()
		for i, wire := range wires {
			oldCharge := currentSimulation.State(wire).Charge()
			charge := newSimulation.State(wire).Charge()
			if oldCharge == charge {
				continue
			}
			position := wire.Bounds().Min
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(position.X), float64(position.Y))
			r, g, b, a := simulationImage.Palette[charge+1].RGBA()
			op.ColorM.Scale(float64(r)/0xFFFF, float64(g)/0xFFFF, float64(b)/0xFFFF, float64(a)/0xFFFF)
			if err := backgroundImage.DrawImage(wireImages[i], op); err != nil {
				return err
			}
		}
		currentSimulation = newSimulation
	default:
	}

	if err := screen.DrawImage(backgroundImage, &ebiten.DrawImageOptions{}); err != nil {
		return err
	}

	if err := handleCursor(screen); err != nil {
		return err
	}
	return nil
}

func drawMask(wire *simulation.Wire) image.Image {
	bounds := image.Rect(0, 0, wire.Bounds().Dx(), wire.Bounds().Dy())
	bounds = bounds.Union(image.Rect(0, 0, 4, 4))
	position := wire.Bounds().Min
	img := image.NewRGBA(bounds)
	white := color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
	for _, pixel := range wire.Pixels() {
		img.SetRGBA(pixel.X-position.X, pixel.Y-position.Y, white)
	}
	return img
}
