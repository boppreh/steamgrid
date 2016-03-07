# What is it? #

Steam has the Big Picture mode and grid view, but they are not perfect. SteamGrid is a small standalone program to
enhance them. It does three things:

- Cache all game images, so you can browse without internet access or that
  annoying delay
- Automagically search, download and set custom images for games that are missing it
- Apply overlays based on your categories

You run it once and it'll set up everything above, automatically. You can run
again when you get more games or want to update the category overlays.

SteamGrid supports Windows and Linux (32 or 64bit), and even non-Steam games.

# Download #

[steamgrid-windows.zip (2.5MB)](https://github.com/boppreh/steamgrid/releases/download/v1.3.0/steamgrid-windows.zip)

[steamgrid-linux.zip (1.9MB)](https://github.com/boppreh/steamgrid/releases/download/v1.3.0/steamgrid-linux.zip)

# How to use #

1. Download the [latest version](https://github.com/boppreh/steamgrid/releases/latest) and extract the zip wherever.
2. *(optional)* Name the overlays after your categories. So if you have a category "Games I Love", put a nice little heart overlay there named "games i love.png". You can rename the defaults that came with the zip or get new ones at [/r/steamgrid](http://www.reddit.com/r/steamgrid/wiki/overlays).
3. Run `steamgrid` and wait. No, really, it's all automatic. Not a single keypress required.
4. Read the report and open Steam in grid view to check the results.

---

[![Results](https://i.imgur.com/QBHZVNx.png)](https://github.com/boppreh/steamgrid/releases/latest)
[![Grid view screenshot](http://i.imgur.com/abnqZ6C.png)](http://i.imgur.com/abnqZ6C.png)
[![Big Picture screenshot](http://i.imgur.com/gv7xDda.png)](http://i.imgur.com/gv7xDda.png)

# Features #

- Grid images are used both in the grid view and Big Picture mode, and SteamGrid works on both.
- Automatically detects Steam installation even in foreign language systems. If
  it still doesn't work for you, just drag and drop the Steam installation folder
  onto the executable for a manual override.
- Detects all local Steam users and customizes their grid images individually.
- Downloads images from two different servers, and falls back to a Google
  search as last resort (don't worry, it'll tell you if that happens).
- Loads your categories from the local Steam installation.
- Applies transparent overlays based on each game categories (make sure the name
  of the overlay file is the name of the category).
- If you already have any customized images it'll use them and apply the
  overlay, but keeping a backup.
- Works just as well with non-Steam games.
- Supports PNG and JPG images.
- Supports games with multiple categories.
- No installation required, just extract the zip and double click.
- Works with Windows and Linux, 32 or 64 bit.
- 100% fire and forget, no interaction required, and can cancel and retry at any moment.

# Something wrong? #

- **Fails to find steam location**: You can drag and drop the Steam installation folder (not the library!) into `steamgrid.exe` for a manual override.
- **A few images were not found**: Some images are hard to find. The program may miss a game, especially betas, prototypes and tests, but you can set an image manually through the Steam client (right click > `Set Custom Image`). Run `steamgrid` again to apply the overlays. If you know a good source of images, drop me a message.
- **No overlays found**: make sure you put your overlays inside the `overlays by category` folder, and it's near the program itself. This error means absolutely no overlays were found, without even taking your categories names into consideration.
- **It didn't apply any overlays**: ensure the overlay file name matches your category name, including possible punctuation (differences in caps are ignored). For example `favorites.png` is used for the `Favorites` category.
- **I'm worried this is a virus**: I work with security, so no offense taken from a little paranoia. The complete source code is provided at this [Github repo](https://github.com/boppreh/steamgrid). If you are worried the binaries don't match the source, you can install Go on your machine and run the sources directly. All it does is save images inside `Steam/userdata/ID/config/grid`. It does connect to the internet, but only to fetch game names from you Steam profile and download images into the Steam's grid image folder. Nothing is installed or saved in the Windows registry, and aside from images downloaded it should leave the computer exactly as it found.

If you encounter any problems please [open an issue](https://github.com/boppreh/steamgrid/issues/new). All critics and suggestions are welcome.
