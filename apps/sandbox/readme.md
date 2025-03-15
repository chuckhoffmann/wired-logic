Wired Logic Playground
======================
This is a playground for Wired Logic. It is where you can draw your own circuits and see them in action.

Controls
--------
Move the mouse or press either the arrow keys or `WASD` to move the cursor.

Clicking the left mouse button or pressing `Space` will flip the state of (i. e. draw or erase) the pixel under the cursor. 

Pressing `P` will toggle the pause state of the simulation.

Pressing `F` will save the current state of the simulation as a GIF image.

Pressing `R` will pause and reset the simulation, setting all pixels to a powered off state. You must press `P` to resume the simulation.

Command line options
--------------------
The playground can be configured using command line options. Any option not specified will use the default value.
- `-width` sets the width of the playground in pixels. Default is 64.
- `-height` sets the height of the playground in pixels. Default is 64.
- `-scale` sets the scale of the playground in pixels per pixel. Default is 16.
- `-speed` sets the speed of the simulation in frames per second. Default is 15

After the options, the path to a GIF image can be specified, e.g. `gif_to_load.gif`. If no path is specified the playground will start with a blank canvas and a default palette. The filename *must* end with `.gif` and appear *after* the other options. Additionally, the height and width of the GIF will override the `-width` and `-height` options.

Here is an example of how to run the playground with custom options and a GIF image:
```
go run main.go -width 32 -height 32 -scale 16 -speed 30 gif_to_load.gif
```
