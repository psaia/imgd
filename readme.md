# imgd

Behind the scenes photography services such as Flickr, Instagram, 500px, Imgur, etc. use cloud providers such as
Google Cloud and Amazon Web Services to store your photos.

`imgd` takes the the middle man out and lets you bring your own cloud provider which puts you in full control of
your photos.

Features:

- Enable photo availability to all of your computers. No need for expensive Apple|Google cloud storage or hard drives.
- imgd will store the photo in full resolution along with creating various sizes
- CLI album management
- Very fast upload/download

* Upload

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

# Generate a sharable gallery page.
imgd album publish silent-escapades --from-template dark

# Download entire album.
imgd album expand silent-escapades

# Remove a gallery and optionally all photos inside. Otherwise,
# photos are added to added to Unknown gallery.
imgd album remove silent-escapades --rm-photos

# Remove a photo by its hash.
imgd photo remove [photo hash]

# View all URLs of photo.
imgd photo show [photo hash]

# Download photo.
imgd photo download [photo hash]

# Remove all galleries and photos in account.
imgd account clean --force
```
