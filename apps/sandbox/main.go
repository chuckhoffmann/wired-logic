package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"log"
	"os"
	"path/filepath"
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
		ebiten.KeyF:     0,
		ebiten.KeyR:     0,
	}
	simulationPaused = false
)

type Config struct {
	speed       int
	scale       int
	width       int
	height      int
	gifFileName string
}

const (
	cursorInitialDelayTicks = 8
	cursorRepeatTicks       = 3
	cursorBlinkCycle        = 128
	cursorBlinkMidpoint     = 64
)

func keyRepeatTriggered(pressCount int) bool {
	if pressCount < 0 {
		return false
	}
	if pressCount == 0 {
		return true
	}
	if pressCount < cursorInitialDelayTicks {
		return false
	}
	return (pressCount-cursorInitialDelayTicks)%cursorRepeatTicks == 0
}

func main() {

	// Parse the command line flags and arguments.
	config, err := parseCommandLineArgs()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Initialize the cursor.
	initializeCursor()

	// Set the cursor position to the center of the screen.
	cursorPosition = image.Point{config.width / 2, config.height / 2}

	// Calculate the time between simulation steps.
	simulationTimer = time.Tick(time.Second / time.Duration(config.speed))

	// If a gif file name is supplied, load the gif file and use the
	// first frame as the simulation image.
	if config.gifFileName != "" {
		in, err := os.Open(config.gifFileName)
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
		// Create a new palette.
		p := color.Palette{
			color.Black,
			color.RGBA{0x88, 0, 0, 0xFF},
			color.RGBA{0xFF, 0, 0, 0xFF},
			color.RGBA{0xFF, 0x22, 0, 0xFF},
			color.RGBA{0xFF, 0x44, 0, 0xFF},
			color.RGBA{0xFF, 0x66, 0, 0xFF},
			color.RGBA{0xFF, 0x88, 0, 0xFF},
			color.RGBA{0xFF, 0xAA, 0, 0xFF},
		}
		// Create a new image for the simulation using the palette.
		simulationImage = image.NewPaletted(image.Rect(0, 0, config.width, config.height), p)
	}
	reloadSimulation()
	if err := ebiten.Run(update, simulationImage.Bounds().Dx(), simulationImage.Bounds().Dy(), float64(config.scale), "Wired Logic"); err != nil {
		log.Fatal(err)
	}
}

func initializeCursor() {
	var err error
	// Create a 4x4 image for the cursor and fill it with white.
	if cursorImage, err = ebiten.NewImage(4, 4, ebiten.FilterNearest); err != nil {
		log.Fatal(err)
	}
	cursorImage.Fill(color.White)
}

func parseCommandLineArgs() (Config, error) {
	var config Config

	flag.IntVar(&config.speed, "speed", 15, "simulation steps per second")
	flag.IntVar(&config.scale, "scale", 16, "pixel scale factor")
	flag.IntVar(&config.width, "width", 64, "width of the simulation")
	flag.IntVar(&config.height, "height", 64, "height of the simulation")
	flag.Parse()

	if config.speed < 1 || config.speed > 60 {
		return config, fmt.Errorf("speed must be between 1 and 60")
	}

	if flag.NArg() > 1 {
		return config, fmt.Errorf("too many arguments")
	}

	if flag.NArg() == 0 {
		return config, nil
	}

	filename := flag.Arg(0)

	if filepath.Ext(filename) != ".gif" {
		return config, fmt.Errorf("file %s does not end with .gif", filename)
	}

	if _, err := os.Stat(filename); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config, fmt.Errorf("file %s does not exist", filename)
		}
		return config, fmt.Errorf("cannot access file %s: %w", filename, err)
	}

	config.gifFileName = filename
	return config, nil
}

func reloadSimulation() error {
	// If there is a change in the simulation image, create a new simulation.
	currentSimulation = simulation.New(simulationImage)
	currentSimulation.Draw(simulationImage)
	var err error
	backgroundImage, err = ebiten.NewImageFromImage(simulationImage, ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}

	// Dispose the old wire images.
	for _, img := range wireImages {
		if err = img.Dispose(); err != nil {
			return err
		}
	}
	// Get the wires from the circuit and create an image for each wire.
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
	for key := range keyStates {
		if !ebiten.IsKeyPressed(key) {
			keyStates[key] = -1
			continue
		}
		keyStates[key]++

	}
}

func handleCursor(screen *ebiten.Image) error {
	// Get the mouse cursor position.
	mx, my := ebiten.CursorPosition()
	mousePosition := image.Point{mx, my}
	mouseMoved := mx != oldMouseCursorPosition.X || my != oldMouseCursorPosition.Y
	cursorMoved := mousePosition.In(screen.Bounds()) && mouseMoved
	oldMouseCursorPosition = mousePosition
	if cursorMoved {
		cursorPosition = oldMouseCursorPosition
	} else {
		switch {
		case keyRepeatTriggered(keyStates[ebiten.KeyUp]) ||
			keyRepeatTriggered(keyStates[ebiten.KeyW]):
			cursorPosition = cursorPosition.Add(image.Point{0, -1})
			cursorMoved = true
		case keyRepeatTriggered(keyStates[ebiten.KeyDown]) ||
			keyRepeatTriggered(keyStates[ebiten.KeyS]):
			cursorPosition = cursorPosition.Add(image.Point{0, +1})
			cursorMoved = true
		case keyRepeatTriggered(keyStates[ebiten.KeyLeft]) ||
			keyRepeatTriggered(keyStates[ebiten.KeyA]):
			cursorPosition = cursorPosition.Add(image.Point{-1, 0})
			cursorMoved = true
		case keyRepeatTriggered(keyStates[ebiten.KeyRight]) ||
			keyRepeatTriggered(keyStates[ebiten.KeyD]):
			cursorPosition = cursorPosition.Add(image.Point{+1, 0})
			cursorMoved = true
		}
	}
	cursorBlinking = (cursorBlinking + 1) % cursorBlinkCycle
	// Set the image draw option for the cursor.
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(0.25, 0.25)
	op.GeoM.Translate(float64(cursorPosition.X), float64(cursorPosition.Y))
	if cursorBlinking > cursorBlinkMidpoint {
		op.ColorM.Scale(1, 1, 1, 0.25+float64(cursorBlinkCycle-1-cursorBlinking)/255.0)
	} else {
		op.ColorM.Scale(1, 1, 1, 0.25+float64(cursorBlinking)/255.0)
	}
	// Draw the cursor.
	if err := screen.DrawImage(cursorImage, op); err != nil {
		return err
	}
	// Toggle the pixel if the space key is pressed or the left mouse button is
	// pressed.
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

	select {
	case <-simulationTimer:
		var newSimulation *simulation.Simulation
		if !simulationPaused {
			newSimulation = currentSimulation.Step()
		} else {
			newSimulation = currentSimulation
		}
		// Draw the wires that have changed.
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
	// Draw the background image.
	if err := screen.DrawImage(backgroundImage, &ebiten.DrawImageOptions{}); err != nil {
		return err
	}
	// Draw the cursor.
	if err := handleCursor(screen); err != nil {
		return err
	}

	// If the P key is pressed, pause/unpause the simulation.
	if keyStates[ebiten.KeyP] == 0 {
		simulationPaused = !simulationPaused

	}

	// if the F key is pressed, save the current simulation to a gif file.
	if keyStates[ebiten.KeyF] == 0 {
		gifFileName := fmt.Sprintf("simulation-%d.gif", time.Now().Unix())
		// TODO: Create a dialog box to get the file name.
		// Save the simulation to the gif file.
		if err := saveImage(simulationImage, gifFileName); err != nil {
			return err
		}
	}

	// If the R key is pressed, pause the simulation, set all
	// pixels that belong to a wire to index 1, and reload the simulation.
	if keyStates[ebiten.KeyR] == 0 {
		simulationPaused = true
		currentSimulation.Draw(simulationImage)
		for _, wire := range currentSimulation.Circuit().Wires() {
			for _, pixel := range wire.Pixels() {
				simulationImage.SetColorIndex(pixel.X, pixel.Y, 1)
			}
		}
		if err := reloadSimulation(); err != nil {
			return err
		}
	}

	return nil
}

func drawMask(wire *simulation.Wire) image.Image {
	// Draw a mask for the wire.
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

func saveImage(img image.Image, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return gif.Encode(f, img, nil)
}
