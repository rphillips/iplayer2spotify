GIT_TAG := $(shell git describe --abbrev=0)
TAG_DISTANCE := $(shell git describe --long | awk -F- '{print $$2}')
PKG_VERSION := ${GIT_TAG}-${TAG_DISTANCE}
GOFILES:=$(shell find . -name '*.go' | grep -v -E '(.vendor)')
LDFLAGS=-X github.com/rphillips/iplayer2spotify/version.Version=${PKG_VERSION} -X github.com/rphillips/iplayer2spotify/secrets.ClientID=${SPOTIFY_ID} -X github.com/rphillips/iplayer2spotify/secrets.SecretKey=${SPOTIFY_SECRET}

build:
	CGO_ENABLED=0 gox \
        	-ldflags="${LDFLAGS}" \
		-osarch="linux/amd64 darwin/amd64 windows/amd64" \
		-output="{{.Dir}}-{{.OS}}-{{.Arch}}"

clean:
	rm -f iplayer2spotify-*

.PHONY: all clean
