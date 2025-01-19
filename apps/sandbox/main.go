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
		ebiten.KeyF:	 0,
		ebiten.KeyR:	 0,
	}
	simulationPaused 		= false
)

func main() {
	var err error
	// Create a 4x4 image for the cursor and fill it with white.
	if cursorImage, err = ebiten.NewImage(4, 4, ebiten.FilterNearest); err != nil {
		log.Fatal(err)
	}
	cursorImage.Fill(color.White)
	// Parse the command line flags, then the gif file name, if supplied.
	var speed, scale, width, height int
	flag.IntVar(&speed, "speed", 15, "simulation steps per second")
	flag.IntVar(&scale, "scale", 16, "pixel scale factor")
	flag.IntVar(&width, "width", 64, "width of the simulation")
	flag.IntVar(&height, "height", 64, "height of the simulation")
	flag.Parse()
	flag.Args()
	// Set the cursor position to the center of the screen.
	cursorPosition = image.Point{width / 2, height / 2}
	// Calculate the time between simulation steps.
	simulationTimer = time.Tick(time.Second / time.Duration(speed))
	// If a gif file name is supplied, load the gif file and use the
	// first frame as the simulation image.
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
		// Create a new palette.
		p := color.Palette{
			color.Black,
			color.RGBA{0x88, 0, 0, 0xFF},
			color.RGBA{0xFF, 0, 0, 0xFF},
			color.RGBA{0xFF, 0x22, 0, 0xFF},
			color.RGBA{0xFF, 0x44, 0, 0xFF},
			color.RGBA{0xFF, 0x66, 0, 0xFF},
			color.RGBA{0xFF, 0x88, 0, 0xFF},
			color.RGBA{0xFF, 0xAA, 0,  0xFF},
		}
		// Create a new image for the simulation. using the palette.
		simulationImage = image.NewPaletted(image.Rect(0, 0, width, height), p)
	}
	reloadSimulation()
	if err := ebiten.Run(update, simulationImage.Bounds().Dx(), simulationImage.Bounds().Dy(), float64(scale), "Wired Logic"); err != nil {
		log.Fatal(err)
	}
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
	for key, _ := range keyStates {
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
	// Set the image draw option for the cursor.
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(0.25, .25)
	op.GeoM.Translate(float64(cursorPosition.X), float64(cursorPosition.Y))
	if cursorBlinking > 64 {
		op.ColorM.Scale(1, 1, 1, 0.25+float64(127-cursorBlinking)/255.0)
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
		// Create a dialog box to get the file name.
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
