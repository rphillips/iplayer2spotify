package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/anaskhan96/soup"
	"github.com/rphillips/iplayer2spotify/secrets"
	"github.com/zmb3/spotify"
)

const redirectURI = "http://localhost:8080/callback"
const maxSongsOnCreate = 100
const maxSearchRetries = 5

var (
	auth  = spotify.NewAuthenticator(redirectURI, spotify.ScopePlaylistModifyPrivate)
	ch    = make(chan *spotify.Client)
	state = stringWithCharset(24, charset)
)

func parseSegments(doc string) []string {
	artistSongs := make([]string, 0)
	parsed := soup.HTMLParse(doc)
	data := parsed.FindAll("div", "class", "p-f")
	for i := 0; i < len(data); i++ {
		if val, ok := data[i].Attrs()["data-title"]; ok {
			artistSongs = append(artistSongs, val)
		}
	}
	return artistSongs
}

func fetchProgramSegments(url string) string {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36")
	body, err := soup.GetWithClient(url, client)
	if err != nil {
		log.Fatal(err)
	}
	return body
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

func searchForSpotifyTracks(client *spotify.Client, artistSongs []string, isCleanOnly bool) ([]spotify.ID, error) {
	songIDs := make([]spotify.ID, 0)
	for _, searchData := range artistSongs {
		log.Printf("Searching for %v\n", searchData)
		splitted := strings.Split(searchData, "||")
		searchStr := fmt.Sprintf("artist:%v %v", splitted[0], splitted[1])
		retry(maxSearchRetries, 5*time.Second, func() error {
			results, err := client.Search(searchStr, spotify.SearchTypeTrack|spotify.SearchTypeArtist)
			if err != nil {
				log.Printf("Spotify search error: %v. Retrying...\n", err)
				return err
			}
			if results.Tracks == nil {
				return nil
			}
			if len(results.Tracks.Tracks) > 0 {
				if isCleanOnly == true && results.Tracks.Tracks[0].Explicit {
					return nil
				}
				songIDs = append(songIDs, results.Tracks.Tracks[0].ID)
			}
			return nil
		})
	}
	return songIDs, nil
}

func main() {
	var showURL = flag.String("show-url", "", "url of show")
	var playlistName = flag.String("playlist-name", "", "playlist name")
	var isCleanOnly = flag.Bool("clean", false, "suppress explicit tracks")

	flag.Parse()

	envStr := os.Getenv("SPOTIFY_ID")
	if envStr != "" {
		secrets.ClientID = envStr
	}
	envStr = os.Getenv("SPOTIFY_SECRET")
	if envStr != "" {
		secrets.SecretKey = envStr
	}
	if secrets.ClientID == "" {
		log.Fatalf("SPOTIFY_ID not set")
	}
	if secrets.SecretKey == "" {
		log.Fatalf("SPOTIFY_SECRET not set")
	}
	if *showURL == "" {
		log.Fatalf("Invalid URL")
	}
	if *playlistName == "" {
		log.Fatalf("Invalid Playlist name")
	}

	endpoint := *showURL + "/segments.inc"
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

	user, err := client.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}

	// Fetch Segments
	doc := fetchProgramSegments(endpoint)
	artistSongs := parseSegments(string(doc))
	songIDs, err := searchForSpotifyTracks(client, artistSongs, *isCleanOnly)
	if err != nil {
		log.Fatal(err)
	}

	// Format Playlist
	now := time.Now()
	nowFormatted := fmt.Sprintf("%02d%02d%02d", now.Year(), now.Month(), now.Day())
	playlistTitle := fmt.Sprintf("%s - %s", *playlistName, nowFormatted)
	if err := createPlaylist(user.ID, client, playlistTitle, songIDs...); err != nil {
		log.Fatal(err)
	}

	log.Printf("Created playlist: %s", playlistTitle)
}
