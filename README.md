Auto Steam Grid
===============


The Steam client has a neat grid view for your games, but it requires continuous internet access, customization has to be done game by game and some games don't have images. Not anymore. This nifty program caches all your game images so you don't need internet access, searches for the ones that Steam is missing and automatically applies customizable overlays based on your categories.

How to use
----------

1. [Download](https://github.com/boppreh/steamgrid/releases/download/v1.1.0-alpha/steamgrid.zip) and extract the zip wherever.
2. *(optional)* Name the overlays after your categories. So if you have a category "Games I Love", put a nice little heart overlay there named "games i love.png". You can rename the defaults that came with the zip or get new ones at [/r/steamgrid](http://www.reddit.com/r/steamgrid/wiki/overlays).
3. Run `steamgrid.exe`, wait a few seconds and close the window. No, really, it's all automatic. Not a single keypress required.
4. Open Steam in grid view and check the results.

[Download here](https://github.com/boppreh/steamgrid/releases/download/v1.1.0-alpha/steamgrid.zip)
---

[![Program screenshot](http://i.imgur.com/QgwSbcq.png)](https://github.com/boppreh/steamgrid/releases/download/v1.1.0-alpha/steamgrid.zip)
[![Steam screenshot with filled grid](http://i.imgur.com/abnqZ6C.png)](https://github.com/boppreh/steamgrid/releases/download/v1.1.0-alpha/steamgrid.zip)


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

Something wrong?
----------------

- **Where's the Linux version?**: It's not well tested, but the [latest release has a Linux version](https://github.com/boppreh/steamgrid/releases/download/v1.0.5/steamgrid-linux.zip). Please report any issues you may have. If you want to run from source, just install Go (`sudo apt-get install golang`) and run `go run steamgrid.go`.
- **Fails to find steam location**: You can drag and drop the Steam installation folder (not the library!) into `steamgrid.exe` for a manual override.
- **A few images were not found**: Some images are hard to find. The program may miss a game, especially betas, prototypes and tests, but you can set an image manually through the Steam client (right click > `Set Custom Image`). Run `steamgrid.exe` again to update the overlays.
- **Can't load profile**: make sure you are connected to the internet and have a [public Steam profile](http://steamcommunity.com/discussions/forum/1/864980009946155418/). If you know how to detect a user's game list without access to their profile, drop me a message.
- **No overlays found**: make sure you put your overlays inside the `overlays by category` folder. This error means absolutely no overlays were found, without even taking your categories names into consideration.
- **It didn't apply any overlays**: ensure the overlay file name matches your category name, including possible punctuation (differences in caps are ignored). If it still fails, it probably means Steam didn't save the categories information at the expected location (`Steam/userdata/[some numbers]/7/remote/sharedconfig.vdf`). Please [open an issue](https://github.com/boppreh/steamgrid/issues/new) to help me fix this problem.
- **I'm worried this is a virus**: I work with security, so no offense taken from a little paranoia. The complete source code is provided at the Github repo. If you are worried the binaries don't match the source, you can install Go on your machine and run the sources directly (see steps above). The only files it touches are the ones inside `Steam/userdata` and the `overlays by category` folder. It does connect to the internet, but only to fetch public Steam profiles and download images into the location above. Nothing is installed or saved in the Windows registry, and aside from images downloaded it should leave the computer exactly as it found.

If you encounter any problems please [open an issue](https://github.com/boppreh/steamgrid/issues/new). All critics and suggestions are welcome.
