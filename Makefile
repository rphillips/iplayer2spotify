all: \
	iplayer2spotify-linux-amd64 \
	iplayer2spotify-windows-amd64 \
	iplayer2spotify-darwin-amd64

GOFILES:=$(shell find . -name '*.go' | grep -v -E '(.vendor)')
LDFLAGS=-X github.com/rphillips/iplayer2spotify/secrets.ClientID=${SPOTIFY_ID} -X github.com/rphillips/iplayer2spotify/secrets.SecretKey=${SPOTIFY_SECRET}

iplayer2spotify-linux-amd64: $(GOFILES)
	@GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $@ .

iplayer2spotify-windows-amd64: $(GOFILES)
	@GOOS=windows GOARCH=amd64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $@ .

iplayer2spotify-darwin-amd64: $(GOFILES)
	@GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $@ .

clean:
	rm -f iplayer2spotify-*

.PHONY: all clean
