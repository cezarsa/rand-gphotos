package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gphotosuploader/googlemirror/api/photoslibrary/v1"
	"golang.org/x/oauth2"
)

var errStop = errors.New("stop")

type authConfig struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	ProjectID    string   `json:"project_id"`
	AuthURI      string   `json:"auth_uri"`
	TokenURI     string   `json:"token_uri"`
	RedirectURIs []string `json:"redirect_uris"`
}

func parseAuth() (*authConfig, error) {
	configEnv := os.Getenv("CONFIG")
	if configEnv == "" {
		return nil, errors.New("required env CONFIG pointing to oauth config file")
	}
	data, err := ioutil.ReadFile(configEnv)
	if err != nil {
		return nil, err
	}
	var a map[string]*authConfig
	err = json.Unmarshal(data, &a)
	if err != nil {
		return nil, err
	}
	return a["installed"], nil
}

func parseToken() (*oauth2.Token, error) {
	_, err := os.Stat(".token")
	if os.IsNotExist(err) {
		return nil, nil
	}
	data, err := ioutil.ReadFile(".token")
	if err != nil {
		return nil, err
	}
	var ret oauth2.Token
	err = json.Unmarshal(data, &ret)
	return &ret, err
}

func saveToken(token *oauth2.Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(".token", data, 0660)
}

func run() error {
	auth, err := parseAuth()
	if err != nil {
		return err
	}
	token, err := parseToken()
	if err != nil {
		return err
	}
	ctx := context.Background()
	oc := oauth2.Config{
		ClientID:     auth.ClientID,
		ClientSecret: auth.ClientSecret,
		Scopes: []string{
			"https://www.googleapis.com/auth/photoslibrary.readonly",
			"https://www.googleapis.com/auth/photoslibrary.sharing",
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:  auth.AuthURI,
			TokenURL: auth.TokenURI,
		},
		RedirectURL: auth.RedirectURIs[0],
	}
	if token == nil {
		url := oc.AuthCodeURL("", oauth2.AccessTypeOffline)
		fmt.Printf("Click here: %s\nPaste code: ", url)
		var code string
		fmt.Scanf("%s", &code)
		token, err = oc.Exchange(ctx, code)
		if err != nil {
			return err
		}
		err = saveToken(token)
		if err != nil {
			return err
		}
	}
	tc := oc.Client(ctx, token)

	cli, err := photoslibrary.New(tc)
	if err != nil {
		return err
	}

	allMedia, err := loadCachedAlbum()
	if err != nil {
		return err
	}

	if allMedia == nil {
		allMedia, err = loadFreshAlbum(ctx, cli)
		if err != nil {
			return err
		}
	}

	for i := 0; i < 20; i++ {
		err = saveRandPhoto(tc, allMedia, i)
		if err != nil {
			return err
		}
	}
	return nil
}

func saveRandPhoto(cli *http.Client, allMedia []*photoslibrary.MediaItem, index int) error {
	rand.Seed(time.Now().Unix())
	mi := randPhoto(allMedia)
	downloadURL := mi.BaseUrl + "=d"
	res, err := cli.Get(downloadURL)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid response: %#v", res)
	}
	outputDir := os.Getenv("OUTPUT_DIR")
	if outputDir == "" {
		outputDir = "."
	}
	outFile, err := os.Create(filepath.Join(outputDir, fmt.Sprintf("img-%04d.jpg", index)))
	if err != nil {
		return err
	}
	_, err = io.Copy(outFile, res.Body)
	if err != nil {
		return err
	}
	return outFile.Close()
}

func loadFreshAlbum(ctx context.Context, cli *photoslibrary.Service) ([]*photoslibrary.MediaItem, error) {
	wantedAlbum := os.Getenv("ALBUM")
	if wantedAlbum == "" {
		return nil, errors.New("required env ALBUM")
	}
	var wantedAlbumID string
	var albumSize int64
	err := cli.Albums.List().Pages(ctx, func(response *photoslibrary.ListAlbumsResponse) error {
		for _, res := range response.Albums {
			if wantedAlbum == res.Title {
				wantedAlbumID = res.Id
				albumSize = res.TotalMediaItems
				return errStop
			}
		}
		return nil
	})
	if err != nil && err != errStop {
		return nil, err
	}
	if wantedAlbumID == "" {
		return nil, errors.New("album not found")
	}
	fmt.Printf("0/%v\n", albumSize)
	var allMedia []*photoslibrary.MediaItem
	err = cli.MediaItems.Search(&photoslibrary.SearchMediaItemsRequest{
		AlbumId:  wantedAlbumID,
		PageSize: 100,
	}).Pages(ctx, func(res *photoslibrary.SearchMediaItemsResponse) error {
		allMedia = append(allMedia, res.MediaItems...)
		fmt.Printf("%v/%v\n", len(allMedia), albumSize)
		return nil
	})
	if err != nil && err != errStop {
		return nil, err
	}
	err = saveAlbum(allMedia)
	return allMedia, err
}

func saveAlbum(media []*photoslibrary.MediaItem) error {
	data, err := json.Marshal(media)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(".album.json", data, 0660)
}

func loadCachedAlbum() ([]*photoslibrary.MediaItem, error) {
	_, err := os.Stat(".album.json")
	if os.IsNotExist(err) {
		return nil, nil
	}
	data, err := ioutil.ReadFile(".album.json")
	if err != nil {
		return nil, err
	}
	var ret []*photoslibrary.MediaItem
	err = json.Unmarshal(data, &ret)
	return ret, err
}

func randPhoto(allMedia []*photoslibrary.MediaItem) *photoslibrary.MediaItem {
	chosen := rand.Intn(len(allMedia))
	for {
		mi := allMedia[chosen]
		if mi.MediaMetadata == nil || mi.MediaMetadata.Photo == nil {
			chosen = (chosen + 1) % len(allMedia)
			continue
		}
		return mi
	}
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}
