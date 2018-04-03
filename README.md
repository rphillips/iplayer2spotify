# iplayer2spotify

Converts a playlist from the BBC Radio to Spotify.

# Usage

Register for a SPOTIFY_ID and SPOTIFY_SECRET
[here](https://developer.spotify.com/my-applications/).

Export the ID and Secret:
```
export SPOTIFY_ID=<your ID>
export SPOTIFY_SECRET=<your secret>
```

```
Usage of ./iplayer2spotify:
  -clean
    	suppress explicit tracks
  -playlist-name string
    	playlist name
  -show-url string
    	url of show
  -username string
    	username
```

# Example

```
./iplayer2spotify \
  -playlist-name "Radio 6's Finest Hour" \
  -show-url "https://www.bbc.co.uk/programmes/b09x8f5t" \
  -username [SPOTIFY_USERNAME]
```
