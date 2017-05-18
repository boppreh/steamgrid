# What is it? #

**SteamGrid** is a standalone, fire-and-forget program to enhance Steam's grid view and Big Picture. It preloads the banner images for all your games (even non-Steam ones) and applies overlays depending on your categories.

You run it once and it'll set up everything above, automatically, keeping your existing custom images. You can run
again when you get more games or want to update the category overlays.

# Download #

[**steamgrid-windows.zip (2.1MB)**](https://github.com/boppreh/steamgrid/releases/download/v2.1.1/steamgrid_windows.zip)

[**steamgrid-linux.zip (2.1MB)**](https://github.com/boppreh/steamgrid/releases/download/v2.1.1/steamgrid_linux.zip)

[**steamgrid-mac.zip (2.2MB)**](https://github.com/boppreh/steamgrid/releases/download/v2.1.1/steamgrid_mac.zip)

# How to use #

1. Download the [latest version](https://github.com/boppreh/steamgrid/releases/latest) and extract the zip wherever.
2. *(optional)* Name the overlays after your categories. So if you have a category "Games I Love", put a nice little heart overlay there named "games i love.png". You can rename the defaults that came with the zip or get new ones at [/r/steamgrid](http://www.reddit.com/r/steamgrid/wiki/overlays).
3. *(optional)* Download a pack of custom images and place it in the `games/` folder. The image files can be either the name of the game (e.g. "Psychonauts.png") or the game id (e.g. "3830.png").
4. Run `steamgrid` and wait. No, really, it's all automatic. Not a single keypress required.
5. Read the report and open Steam in grid view to check the results.

---

[![Results](https://i.imgur.com/HiBCe7p.png)](https://i.imgur.com/HiBCe7p.png)
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
- If a game is missing an official banner *and* a name (common for prototypes), it gets the name
  from SteamDB and google searches the banner.
- Loads your categories from the local Steam installation.
- Applies transparent overlays based on each game categories (make sure the name
  of the overlay file is the name of the category).
- If you already have any customized images it'll use them and apply the
  overlay, but keeping a backup.
- If you have images in the directory `games/`, it'll search by game name or by id and use them.
- Works just as well with non-Steam games.
- Supports PNG and JPG images.
- Supports games with multiple categories.
- No installation required, just extract the zip and double click.
- Works with Windows, Linux, and MacOS, 32 or 64 bit.
- 100% fire and forget, no interaction required, and can cancel and retry at any moment.

# Something wrong? #

- **Why are there crowns and other icons on top of my images?**: Those are the default overlays for categories, found in the folder `overlays by category/`. You can download new ones, or just delete the file and re-run SteamGrid to remove the overlay.
- **Fails to find steam location**: You can drag and drop the Steam installation folder (not the library!) into `steamgrid.exe` for a manual override.
- **A few images were not found**: Some images are hard to find. The program may miss a game, especially betas, prototypes and tests, but you can set an image manually through the Steam client (right click > `Set Custom Image`). Run `steamgrid` again to apply the overlays. If you know a good source of images, drop me a message.
- **No overlays found**: make sure you put your overlays inside the `overlays by category` folder, and it's near the program itself. This error means absolutely no overlays were found, without even taking your categories names into consideration.
- **It didn't apply any overlays**: ensure the overlay file name matches your category name, including possible punctuation (differences in caps are ignored). For example `favorites.png` is used for the `Favorites` category.
- **I'm worried this is a virus**: I work with security, so no offense taken from a little paranoia. The complete source code is provided at this [Github repo](https://github.com/boppreh/steamgrid). If you are worried the binaries don't match the source, you can install Go on your machine and run the sources directly. All it does is save images inside `Steam/userdata/ID/config/grid`. It does connect to the internet, but only to fetch game names from you Steam profile and download images into the Steam's grid image folder. Nothing is installed or saved in the Windows registry, and aside from images downloaded it should leave the computer exactly as it found.

If you encounter any problems please [open an issue](https://github.com/boppreh/steamgrid/issues/new). All critics and suggestions are welcome.
