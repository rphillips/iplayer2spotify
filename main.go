package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/anaskhan96/soup"
	"github.com/rphillips/iplayer2spotify/secrets"
	"github.com/rphillips/iplayer2spotify/version"
	"github.com/zmb3/spotify"
	"gopkg.in/urfave/cli.v1"
)

const defaultDateFormat = "20060102"
const maxSearchRetries = 5
const maxSongsOnCreate = 100
const redirectURI = "http://localhost:8080/callback"
const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36"

var (
	auth  = spotify.NewAuthenticator(redirectURI, spotify.ScopePlaylistModifyPrivate)
	ch    = make(chan *spotify.Client)
	state = stringWithCharset(24, charset)
)

func parseSegments(doc string) []string {
	artistSongs := make([]string, 0)
	parsed := soup.HTMLParse(doc)
	segments := parsed.FindAll("div", "class", "segment__track")
	for _, node := range segments {
		artistSongSegments := node.FindAll("span")
		if len(artistSongSegments) != 2 {
			continue
		}
		artist := artistSongSegments[0].Text()
		song := artistSongSegments[1].Text()
		artistSongs = append(artistSongs, fmt.Sprintf("%s - %s", artist, song))
	}
	return artistSongs
}

func fetchProgramSegments(url string) (string, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("User-Agent", userAgent)
	return soup.GetWithClient(url, client)
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}
	// use the token to get an authenticated client
	client := auth.NewClient(tok)
	fmt.Fprintf(w, "Login Completed!")
	ch <- &client
}

func createPlaylist(username string, client *spotify.Client, title string, songIDs ...spotify.ID) error {
	playlist, err := client.CreatePlaylistForUser(username, title, false)
	if err != nil {
		return err
	}
	if err := client.ReplacePlaylistTracks(username, playlist.ID, songIDs...); err != nil {
		return err
	}
	// Support playlists greater than 100 tracks
	if len(songIDs) > maxSongsOnCreate {
		songIDs = songIDs[maxSongsOnCreate:]
		for len(songIDs) > 0 {
			if _, err := client.AddTracksToPlaylist(username, playlist.ID, songIDs...); err != nil {
				return err
			}
			songIDs = songIDs[maxSongsOnCreate:]
		}
	}
	return nil
}

func createPlaylistTitle(opts *Options) (string, error) {
	tmplData := struct {
		Now       string
		CleanOnly bool
	}{
		time.Now().Format(opts.DateFormat),
		opts.CleanOnly,
	}
	tmpl, err := template.New("playlist_title").Parse(opts.PlaylistNameTmpl)
	if err != nil {
		return "", err
	}
	var playlistTitle bytes.Buffer
	if err := tmpl.Execute(&playlistTitle, tmplData); err != nil {
		return "", err
	}
	return playlistTitle.String(), nil
}

func searchForSpotifyTracks(opts *Options, client *spotify.Client, artistSongs []string) ([]spotify.ID, error) {
	songIDs := make([]spotify.ID, 0)
	for _, searchData := range artistSongs {
		log.Printf("Searching for %v\n", searchData)
		retry(maxSearchRetries, 5*time.Second, func() error {
			results, err := client.Search(searchData, spotify.SearchTypeTrack|spotify.SearchTypeArtist)
			if err != nil {
				log.Printf("Spotify search error: %v. Retrying...\n", err)
				return err
			}
			if results.Tracks == nil {
				log.Printf("Could not find %v", searchData)
				return nil
			}
			if len(results.Tracks.Tracks) > 0 {
				if opts.CleanOnly == true && results.Tracks.Tracks[0].Explicit {
					log.Printf("Explicit song rejected %v", searchData)
					return nil
				}
				songIDs = append(songIDs, results.Tracks.Tracks[0].ID)
			}
			return nil
		})
	}
	return songIDs, nil
}

type Options struct {
	ShowURL          string
	PlaylistNameTmpl string
	CleanOnly        bool
	DateFormat       string
}

func run(c *cli.Context) error {
	options := &Options{
		ShowURL:          c.String("show-url"),
		PlaylistNameTmpl: c.String("playlist-name"),
		CleanOnly:        c.Bool("clean"),
		DateFormat:       c.String("date-format"),
	}

	if s := c.String("spotify-id"); s != "" {
		secrets.ClientID = s
	}
	if s := c.String("spotify-secret"); s != "" {
		secrets.SecretKey = s
	}
	if secrets.ClientID == "" {
		return errors.New("SPOTIFY_ID not set")
	}
	if secrets.SecretKey == "" {
		return errors.New("SPOTIFY_SECRET not set")
	}
	if options.ShowURL == "" {
		return errors.New("Invalid URL")
	}
	if options.PlaylistNameTmpl == "" {
		return errors.New("Invalid Playlist Template")
	}
	// Validate playlist template name
	playlistTitle, err := createPlaylistTitle(options)
	if err != nil {
		return err
	}

	endpoint := options.ShowURL
	log.Printf("Using Show URL %v\n", endpoint)

	// Setup auth callback
	http.HandleFunc("/callback", completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})
	go http.ListenAndServe(":8080", nil)

	// Auth to Spotify
	auth.SetAuthInfo(secrets.ClientID, secrets.SecretKey)
	url := auth.AuthURL(state)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)

	// wait for auth to complete
	client := <-ch
	client.AutoRetry = true

	// Fetch current user
	user, err := client.CurrentUser()
	if err != nil {
		return err
	}

	// Fetch Segments
	doc, err := fetchProgramSegments(endpoint)
	if err != nil {
		return err
	}
	artistSongs := parseSegments(doc)
	songIDs, err := searchForSpotifyTracks(options, client, artistSongs)
	if err != nil {
		return err
	}

	// Format Playlist
	if err := createPlaylist(user.ID, client, playlistTitle, songIDs...); err != nil {
		return err
	}
	log.Printf("Created playlist: %s", playlistTitle)
	return nil
}

func main() {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("%s\n", c.App.Version)
	}
	app := cli.NewApp()
	app.Name = "iplayer2spotify"
	app.Usage = "iplayer (BBC) playlist to Spotify converter"
	app.Action = run
	app.Version = version.Version
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "clean",
			Usage: "suppress explicit tracks",
		},
		cli.StringFlag{
			Name:  "playlist-name",
			Usage: "playlist name, allows for go template (Supported Variables: {{ .Now }} {{ .CleanOnly }}",
		},
		cli.StringFlag{
			Name:   "spotify-id",
			Usage:  "Developer ID",
			EnvVar: "SPOTIFY_ID",
			Hidden: true,
		},
		cli.StringFlag{
			Name:   "spotify-secret",
			Usage:  "Developer Secret",
			EnvVar: "SPOTIFY_SECRET",
			Hidden: true,
		},
		cli.StringFlag{
			Name:  "show-url",
			Usage: "url of show",
		},
		cli.StringFlag{
			Name:  "date-format",
			Usage: "date format to use",
			Value: defaultDateFormat,
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
