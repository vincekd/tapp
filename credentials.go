package tapp

import (
	"os"
	"io/ioutil"
	"encoding/json"
	"github.com/ChimeraCoder/anaconda"
)

type Credentials struct {
	RequestToken string `json:"requestToken"`
	RequestTokenSecret string `json:"requestTokenSecret"`
	AccessToken string `json:"accessToken"`
	AccessTokenSecret string `json:"accessTokenSecret"`
	ConsumerKey string `json:"consumerKey"`
	ConsumerKeySecret string `json:"consumerKeySecret"`
	ScreenName string `json:"screenName"`
	GaKey string `json:"gaTrackingId"`
}

func LoadCredentials() (api *anaconda.TwitterApi, token Credentials) {
	credentials, err := ioutil.ReadFile("credentials")
	if err != nil {
		os.Exit(1)
	}

	err = json.Unmarshal([]byte(credentials), &token)
	if err != nil {
		os.Exit(1)
	}

	anaconda.SetConsumerKey(token.ConsumerKey)
	anaconda.SetConsumerSecret(token.ConsumerKeySecret)
	//api = anaconda.NewTwitterApi(token.AccessToken, token.AccessTokenSecret)
	api = anaconda.NewTwitterApi("", "")

	return api, token
}
