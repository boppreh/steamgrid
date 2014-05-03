Auto Steam Grid
===============

The Steam client has a neat grid view for your games, but it requires
continuous internet access and customization has to be done game by game.

![Steam screenshot with filled grid](http://i.imgur.com/abnqZ6C.png)

Not anymore. This nifty program caches all your game images so you don't need
internet access, searches for the ones that Steam is missing and it also
applies overlays based on your categories.

Just extract it anywhere, customize the overlays to your liking (or leave the
defaults) and run `steamgrid.exe`. It'll automatically detect your Steam
installation, local users, their games and categories and work all the magic.
Not a single keypress required.

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
