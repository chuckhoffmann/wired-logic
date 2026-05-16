package main

import (
	"flag"
	"image"
	"image/color"
	"image/gif"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestKeyRepeatTriggered(t *testing.T) {
	tests := []struct {
		name       string
		pressCount int
		want       bool
	}{
		{name: "negative count", pressCount: -1, want: false},
		{name: "initial press triggers", pressCount: 0, want: true},
		{name: "before initial delay does not trigger", pressCount: 1, want: false},
		{name: "just before delay end does not trigger", pressCount: cursorInitialDelayTicks - 1, want: false},
		{name: "at delay boundary triggers", pressCount: cursorInitialDelayTicks, want: true},
		{name: "between repeat ticks does not trigger", pressCount: cursorInitialDelayTicks + 1, want: false},
		{name: "next repeat tick triggers", pressCount: cursorInitialDelayTicks + cursorRepeatTicks, want: true},
		{name: "later repeat tick triggers", pressCount: cursorInitialDelayTicks + 2*cursorRepeatTicks, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := keyRepeatTriggered(tc.pressCount)
			if got != tc.want {
				t.Fatalf("keyRepeatTriggered(%d) = %v, want %v", tc.pressCount, got, tc.want)
			}
		})
	}
}

func TestCreateNewSimulationImage(t *testing.T) {
	img, err := createNewSimulationImage(3, 2)
	if err != nil {
		t.Fatalf("createNewSimulationImage returned error: %v", err)
	}

	if got, want := img.Bounds(), image.Rect(0, 0, 3, 2); got != want {
		t.Fatalf("bounds = %v, want %v", got, want)
	}

	if got, want := len(img.Palette), 8; got != want {
		t.Fatalf("palette length = %d, want %d", got, want)
	}

	if got := img.Palette[0]; got != color.Black {
		t.Fatalf("palette[0] = %v, want black", got)
	}

	if got := img.ColorIndexAt(0, 0); got != 0 {
		t.Fatalf("ColorIndexAt(0, 0) = %d, want 0", got)
	}

}

func TestParseCommandLineArgs(t *testing.T) {
	originalCommandLine := flag.CommandLine
	originalArgs := os.Args
	t.Cleanup(func() {
		flag.CommandLine = originalCommandLine
		os.Args = originalArgs
	})

	t.Run("defaults", func(t *testing.T) {
		flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
		flag.CommandLine.SetOutput(io.Discard)
		os.Args = []string{"sandbox"}

		got, err := parseCommandLineArgs()
		if err != nil {
			t.Fatalf("parseCommandLineArgs returned error: %v", err)
		}

		if got.speed != 15 || got.scale != 16 || got.width != 64 || got.height != 64 || got.gifFileName != "" {
			t.Fatalf("unexpected defaults: %+v", got)
		}
	})

	t.Run("rejects invalid speed", func(t *testing.T) {
		flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
		flag.CommandLine.SetOutput(io.Discard)
		os.Args = []string{"sandbox", "-speed", "0"}

		_, err := parseCommandLineArgs()
		if err == nil {
			t.Fatal("parseCommandLineArgs returned nil error, want failure")
		}
	})

	t.Run("accepts gif file argument", func(t *testing.T) {
		gifPath := writeTestGIF(t)
		flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
		flag.CommandLine.SetOutput(io.Discard)
		os.Args = []string{"sandbox", gifPath}

		got, err := parseCommandLineArgs()
		if err != nil {
			t.Fatalf("parseCommandLineArgs returned error: %v", err)
		}

		if got.gifFileName != gifPath {
			t.Fatalf("gifFileName = %q, want %q", got.gifFileName, gifPath)
		}
	})
}

func TestCreateSimulationImageFromGif(t *testing.T) {
	gifPath := writeTestGIF(t)

	img, err := createSimulationImageFromGif(gifPath)
	if err != nil {
		t.Fatalf("createSimulationImageFromGif returned error: %v", err)
	}

	if got, want := img.Bounds(), image.Rect(0, 0, 2, 2); got != want {
		t.Fatalf("bounds = %v, want %v", got, want)
	}

	if got := img.Palette[0]; got != color.Transparent {
		t.Fatalf("palette[0] = %v, want transparent", got)
	}
}

func writeTestGIF(t *testing.T) string {
	t.Helper()

	img := image.NewPaletted(image.Rect(0, 0, 2, 2), color.Palette{
		color.RGBA{0x11, 0x22, 0x33, 0xFF},
		color.RGBA{0x44, 0x55, 0x66, 0xFF},
	})
	img.SetColorIndex(0, 0, 1)

	path := filepath.Join(t.TempDir(), "sample.gif")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create temp gif: %v", err)
	}
	if err := gif.Encode(f, img, nil); err != nil {
		f.Close()
		t.Fatalf("encode gif: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close gif: %v", err)
	}

	return path
}
