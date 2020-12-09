# imgd

`imgd` is a BYOC (bring your own cloud) utility to store, host, and manage your photos. This is an alternative to
paying for a service like Drive, Dropbox, or Adobe.

## Features

- Easily store the original/full resolution image while stile having web-friendly versions to share
- Static gallery generation with customizable [themes](templates/limpo)
- It's fast
- Easily manage your images from multiple computers because the state/db is stored in the same bucket as the images

## Usage

1. Obtain a [service account JSON key](https://console.cloud.google.com/iam-admin/serviceaccounts/create) from Google Cloud (currently GCS is only supported) with a "Storage Admin" role
2. Set some environmental variables in your environment:

```bash
# These can alternatively be passed in as flags to the CLI tool, but this is easier.
export IMGD_PROVIDER=gcs # Currently, "gcs" is the only provider available.
export IMGD_GCS_CREDENTIALS="${HOME}/Desktop/your-service-account-key.json"

# This may be useful to set to 1 if you're working with HUGE files. Set to something like 10
# if you're working with many smaller files. It defaults to the number of CPUs you have.
# export CONCURRENCY=1

# This may be useful if you're doing development.
# export DEBUG=1
```

3. `cd` into the imgd source code directory and run `make build`
4. You can now use `./imgd ...`

```bash
# Create a new directory or update if already exists.
imgd album create \
    --title="Silent Escapades in San Francisco" \
    --description="Something about the gallery."

# List all albums and obtain album IDs.
imgd album list

# Upload all photos within a given directory. Only photos that have been added are removed are synced.
# This also regenerates all static html files regardless of what has been removed or added.
imgd album sync ALBUM_ID ./folder-with-photos

# List all photos in album.
imgd album expand ALBUM_ID

# Download entire album to a new directory.
imgd album download ALBUM_ID ./my-gallery

# Remove a gallery of photos.
imgd album remove ALBUM_ID

######################################################
## Below are still WIP:

# Remove a photo by its hash.
imgd photo remove [photo hash]

# View all URLs of photo.
imgd photo show [photo hash]

# Download photo.
imgd photo download [photo hash]

# Remove all galleries and photos in account.
imgd account clean --force
```

## TODO

- Include binaries to make it easier to get started (and add to brew)
- Video
- More CLI options
- An API for a desktop client
- A desktop client
