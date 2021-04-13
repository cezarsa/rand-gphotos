package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/gphotosuploader/googlemirror/api/photoslibrary/v1"
	"golang.org/x/oauth2"
)

type authConfig struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	ProjectID    string   `json:"project_id"`
	AuthURI      string   `json:"auth_uri"`
	TokenURI     string   `json:"token_uri"`
	RedirectURIs []string `json:"redirect_uris"`
}

func parseAuth() (*authConfig, error) {
	data, err := ioutil.ReadFile(os.Getenv("CONFIG"))
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
	cli.Albums.List().Pages(ctx, func(response *photoslibrary.ListAlbumsResponse) error {
		for _, res := range response.Albums {
			fmt.Printf("album: %s - %s\n", res.Id, res.Title)
		}
		return nil
	})
	return nil
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}
