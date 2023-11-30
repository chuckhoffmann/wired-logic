Wired Logic Playground
======================
This is a playground for Wired Logic. It is where you can draw your own circuits and see them in action.

Controls
--------
Move the mouse or press either the arrow keys or `WASD` to move the cursor.

Clicking the left mouse button or pressing `Space` will flip the state of the pixel under the cursor.

Pressing `P` will toggle the pause state of the simulation.

Pressing `F` will save the current state of the simulation as a GIF image.

Command line options
--------------------
The playground can be configured using command line options.
- `-width` sets the width of the playground in pixels.
- `-height` sets the height of the playground in pixels.
- `-scale` sets the scale of the playground in pixels per pixel.
- `-speed` sets the speed of the simulation in frames per second.
After the options, the path to a GIF image can be specified, e.g. `gif_to_load.gif`. If no path is specified the playground will start with a blank canvas.