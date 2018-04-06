# iplayer2spotify [![Build Status](https://travis-ci.org/rphillips/iplayer2spotify.svg?branch=master)](https://travis-ci.org/rphillips/iplayer2spotify)

Converts a playlist from the [BBC Radio](http://www.bbc.co.uk/radio/schedules) to Spotify.

# Install

Use a release version!

# Usage

```
Usage of ./iplayer2spotify:
  -clean
    	suppress explicit tracks
  -playlist-name string
    	playlist name
  -show-url string
    	url of show
```

# Example

```
./iplayer2spotify \
  -playlist-name "Radio 6's Finest Hour" \
  -show-url "https://www.bbc.co.uk/programmes/b09x8f5t"
```

# Using your own Spotify Secrets

Export the ID and Secret:
```
export SPOTIFY_ID=<your ID>
export SPOTIFY_SECRET=<your secret>
```

