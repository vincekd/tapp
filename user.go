package tapp

import (
	"context"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/datastore"
)

type User struct {
	ScreenName string
	Id int64
	Url string
	ProfileImageUrlHttps string
	Name string
	Description string
	Followers int
	Following int
	TweetCount int64
	Location string
	Verified bool
	Link string
	Updated int64
	Media Media
}

func (user User) GetKey(ctx context.Context) *datastore.Key {
	return datastore.NewKey(ctx, "User", user.ScreenName, 0, nil)
}

func (user User) Store(ctx context.Context) error {
	newKey, err := datastore.Put(ctx, user.GetKey(ctx), &user)
	if err != nil {
		log.Errorf(ctx, "Error storing user in db: %v", err)
		return err
	} else {
		log.Infof(ctx, "Stored user: %v", newKey)
	}
	return nil
}
