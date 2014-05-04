Auto Steam Grid
===============


The Steam client has a neat grid view for your games, but it requires
continuous internet access and customization has to be done game by game.

Not anymore. This nifty program caches all your game images so you don't need
internet access, searches for the ones that Steam is missing and it also
applies overlays based on your categories.

How to use
----------

1. [Download](https://github.com/boppreh/steamgrid/releases/download/v1.0.1/steamgrid.zip) and extract the zip wherever.
2. *(optional)* Name the overlays after your categories. So if you have a category "Games I Love", put a nice little heart overlay there named "games i love.png". You can rename the defaults that came with the zip or get new ones at [/r/steamgrid](http://www.reddit.com/r/steamgrid/wiki/overlays).
3. Run `steamgrid.exe` and wait a few seconds. No, really, it's all automatic. Not a single keypress required.
4. Open Steam in grid view and check the results.

If the program could not find some image, which may happen with betas, prototypes and test games, you can set the image manually through the Steam client (right click > Set Custom Image). Run `steamgrid.exe` again to update the overlays.

[Download zip here](https://github.com/boppreh/steamgrid/releases/download/v1.0.1/steamgrid.zip)
---

[![Steam screenshot with filled grid](http://i.imgur.com/abnqZ6C.png)](https://github.com/boppreh/steamgrid/releases/download/v1.0.1/steamgrid.zip)


Features
--------

- Automatically detects Steam installation even in foreign language systems. If
  it doesn't work for some reason with you, just drag and drop the steam folder
  onto the executable.
- Detects all local Steam users and customizes their grid images individually.
- Loads your game list from your public Steam profile (make sure you have one!)
- Downloads grid images from two different servers, and falls back to a Google
  search as last resort (don't worry, it'll tell you if this happens).
- Loads your categories from the Steam installation.
- Applies transparent overlays based on the game category (make sure the name
  of the overlay file is the name of the category).
- If you already have any customized images it'll use them and apply the
  overlay, but keeping a backup.
- 100% fire and forget, no interaction required.

If you encounter any problems please open an issue or email me. All critics and
suggestions welcome.
