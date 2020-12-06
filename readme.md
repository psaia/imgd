# imgd

`imgd` is a BYOC (bring your own cloud) utility to store, host, and manage your photos.

## Features

- Really fast

## Usage

```bash

# Create a new directory or update if already exists.
imgd album create silent-escapades
    --title="Silent Escapades in San Francisco" \
    --description="Something about the gallery."

# List all albums.
imgd album list

# Upload all photos within given directory.
imgd album sync silent-escapades ./folder-with-photos

# List all photos in album.
imgd album expand silent-escapades

# Download entire album to a new directory.
imgd album expand silent-escapades ./my-gallery

# Remove a gallery of photos.
imgd album remove silent-escapades

######################################################
## Below are still WIP:

# Generate a sharable gallery page.
imgd album publish silent-escapades --from-template dark

# Remove a photo by its hash.
imgd photo remove [photo hash]

# View all URLs of photo.
imgd photo show [photo hash]

# Download photo.
imgd photo download [photo hash]

# Remove all galleries and photos in account.
imgd account clean --force
```
