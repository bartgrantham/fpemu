
## Next

* draw the CPU, etc. state much better
* abstract CPU and MEM into a "board" that can be started
    * add in the PIA
* need: breakpoints, "dirty" flags for what's changed between displays
* free-running CPU with ticks on channel if not free-running
    * probably best to implement this as cycles/frame, with frames every 1/60s or so
* keydown event loop for debugger, breakpoints on/off, continue, 1s/.1s/.01s/.001s/full speeds, and most importantly triggering a sound!
