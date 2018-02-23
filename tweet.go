package tapp

import (
	"strings"
	"context"
	"google.golang.org/appengine/datastore"
)

type MyTweet struct {
	Id int64
	IdStr string
	ReplyTo int64
	Created int64
	Updated int64

	Faves int
	Rts int
	Ratio float32

	Text string
	Url string
	Deleted bool
	Media []Media
}

func (tweet MyTweet) GetKey(ctx context.Context) *datastore.Key {
	return datastore.NewKey(ctx, "MyTweet", "", tweet.Id, nil)
}

func (tweet MyTweet) MatchesTerms(terms [][]SearchTerm) bool {
	text := strings.ToUpper(RemovePunctuation(tweet.Text, true))
	for _, or := range terms {
		match := true
		for _, term := range or {
			if term.Quoted == true && term.RegExp != nil {
				if term.RegExp.MatchString(text) == false {
					match = false
					break
				}
			} else {
				if strings.Contains(text, term.Upper) == false {
					match = false
					break
				}
			}
		}
		if match == true {
			return true
		}
	}
	return false
}
