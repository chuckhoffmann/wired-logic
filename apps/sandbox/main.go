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
		ebiten.KeyZ:     0,
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

func main() {

	// Parse the command line flags and arguments.
	config, err := parseCommandLineArgs()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize the cursor.
	initializeCursor()

	// Set the cursor position to the center of the screen.
	cursorPosition = image.Point{config.width / 2, config.height / 2}

	// Calculate the time between simulation steps.
	simulationTimer = time.Tick(time.Second / time.Duration(config.speed))

	// If a gif file name is supplied, create the simulation image from
	// the gif file. Otherwise, create a new simulation image.
	if config.gifFileName != "" {
		simulationImage, err = createSimulationImageFromGif(config.gifFileName)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		simulationImage, err = createNewSimulationImage(config.width, config.height)
		if err != nil {
			log.Fatal(err)
		}
	}

	if err := reloadSimulation(); err != nil {
		log.Fatal(err)
	}

	if err := ebiten.Run(update, simulationImage.Bounds().Dx(), simulationImage.Bounds().Dy(), float64(config.scale), "Wired Logic"); err != nil {
		log.Fatal(err)
	}
}

// parseCommandLineArgs parses the command line flags and arguments and returns a Config struct. The flags are:
// -speed: the simulation steps per second (default 15, must be between 1 and 60)
// -scale: the pixel scale factor (default 16)
// -width: the width of the simulation (default 64)
// -height: the height of the simulation (default 64)
//
// The arguments are:
// [0]: an optional gif file name to load as the initial simulation state. If no file name is provided, a new simulation image will be created with the specified width and height. The file must end with .gif and must exist.
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

// initializeCursor creates a 4x4 white image for the cursor. The cursor will be drawn at the mouse position and will blink by changing its alpha value over time.
func initializeCursor() {
	var err error
	// Create a 4x4 image for the cursor and fill it with white.
	if cursorImage, err = ebiten.NewImage(4, 4, ebiten.FilterNearest); err != nil {
		log.Fatal(err)
	}
	cursorImage.Fill(color.White)
}

// createSimulationImageFromGif creates a simulation image from the first frame of the given gif file. 
func createSimulationImageFromGif(filename string) (*image.Paletted, error) {
	//
	in, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open gif %s: %w", filename, err)
	}
	defer in.Close()

	gifImage, err := gif.DecodeAll(in)
	if err != nil {
		return nil, fmt.Errorf("decode gif %s: %w", filename, err)
	}

	if len(gifImage.Image) == 0 {
		return nil, fmt.Errorf("gif %s contains no frames", filename)
	}
	// Get the first frame of the gif. If the first frame has a palette, set the first color to transparent. This allows the user to create a gif with a transparent background by setting the first color in the palette to transparent.
	firstFrame := gifImage.Image[0]

	if len(firstFrame.Palette) > 0 {
		firstFrame.Palette[0] = color.Transparent
	}

	return firstFrame, nil
}

// createNewSimulationImage creates a new simulation image with the given width and height, and a predefined palette. The palette has 9 colors: transparent, black, and 7 shades from red to yellow. The image is initialized with all pixels set to transparent.
func createNewSimulationImage(width, height int) (*image.Paletted, error) {
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
	return image.NewPaletted(image.Rect(0, 0, width, height), p), nil
}

// reloadSimulation creates a new simulation from the current simulation image and redraws the background and wire images. This is called when the user toggles a pixel or resets the simulation to the wire seed. It returns an error if any of the image operations fail.
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

// togglePixel toggles the pixel at the given position in the simulation image. If the pixel is currently transparent, it will be set to black (color index 1). If the pixel is currently black or any of the red shades (color index 1-8), it will be set to transparent (color index 0). After toggling the pixel, it calls reloadSimulation to update the simulation and redraw the images. It returns an error if any of the image operations fail.
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
	if err := applyHotkeys(); err != nil {
		return err
	}

	select {
	case <-simulationTimer:
		if !simulationPaused {
			var err error
			currentSimulation, err = stepSimulationAndRedraw()
			if err != nil {
				return err
			}
		}

	default:

	}
	// Draw the background image on the screen.
	if err := screen.DrawImage(backgroundImage, &ebiten.DrawImageOptions{}); err != nil {
		return err
	}
	// Draw the cursor.
	if err := handleCursor(screen); err != nil {
		return err
	}

	return nil
}

func stepSimulationAndRedraw() (*simulation.Simulation, error) {
	var newSimulation *simulation.Simulation
	newSimulation = currentSimulation.Step()
	// Draw the wires that have changed.
	wires := currentSimulation.Circuit().Wires()
	for i, wire := range wires {
		oldCharge := currentSimulation.State(wire).Charge()
		charge := newSimulation.State(wire).Charge()
		if oldCharge == charge {
			continue
		}
		// Get the position of the wire and draw the corresponding wire image with the color corresponding to the new charge.
		position := wire.Bounds().Min
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(position.X), float64(position.Y))
		r, g, b, a := simulationImage.Palette[charge+1].RGBA()
		op.ColorM.Scale(float64(r)/0xFFFF, float64(g)/0xFFFF, float64(b)/0xFFFF, float64(a)/0xFFFF)
		if err := backgroundImage.DrawImage(wireImages[i], op); err != nil {
			return nil, err
		}
	}

	return newSimulation, nil
}

// applyHotkeys checks the key states for the hotkeys and performs the corresponding actions. The hotkeys are:
// - P: pause/unpause the simulation on key-down edge.
// - F: export a snapshot of the current simulation state as a gif file on key-down edge.
// - R: reset the simulation on key-down edge.	
func applyHotkeys() error {
	// Pause/unpause simulation on key-down edge.
	if keyStates[ebiten.KeyP] == 0 {
		simulationPaused = !simulationPaused
	}

	// Export snapshot on key-down edge.
	if keyStates[ebiten.KeyF] == 0 {
		if err := exportSnapshot(); err != nil {
			return err
		}
	}

	// Reset wires on key-down edge.
	if keyStates[ebiten.KeyR] == 0 {
		if err := resetSimulationToWireSeed(); err != nil {
			return err
		}
	}

	if keyStates[ebiten.KeyZ] == 0 {
	}

	return nil
}

func exportSnapshot() error {
	// Materialize current state into the palette image before saving.
	currentSimulation.Draw(simulationImage)
	gifFileName := fmt.Sprintf("simulation-%s.gif", time.Now().Format("2006-01-02-150405"))
	return saveImage(simulationImage, gifFileName)
}

func resetSimulationToWireSeed() error {
	simulationPaused = true

	currentSimulation.Draw(simulationImage)
	for _, wire := range currentSimulation.Circuit().Wires() {
		for _, pixel := range wire.Pixels() {
			simulationImage.SetColorIndex(pixel.X, pixel.Y, 1)
		}
	}

	return reloadSimulation()
}

// drawMask creates a mask image for the given wire. The mask is a white image with the same dimensions as the wire's bounding box, where the pixels corresponding to the wire's pixels are set to white and the rest are transparent. This mask is used to draw the wire on the background image with the correct color corresponding to its charge.
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

// saveImage saves the given image to a file with the given filename in gif format. It returns an error if any of the file operations fail.
func saveImage(img image.Image, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return gif.Encode(f, img, nil)
}
