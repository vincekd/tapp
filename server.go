package main

import (
	"os"
	"fmt"
	"time"
	"math"
	"strings"
	"regexp"
	"context"
	"path"
	"net/http"
	"net/url"
	"strconv"
	"html/template"
	"encoding/json"
	"encoding/csv"
	"io/ioutil"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/memcache"
	"google.golang.org/appengine/datastore"
	"github.com/ChimeraCoder/anaconda"
)

var (
	twitterApi *anaconda.TwitterApi
	screenName string
)
const (
	TWITTER_URL = "https://twitter.com/"
	MEMCACHE_TWEET_KEY = "TWEETS"
	MEMCACHE_USER_KEY = "USER."
	TWEETS_TO_FETCH = 30
	MIN_RATIO = float32(0.10)
	MAX_PUT_SIZE int = 500
	MAX_API_LOOKUP_SIZE int = 100
	MIN_SEARCH_LENGTH int = 2
	SEARCH_TIME_FORMAT = "Mon Jan 2 15:04:05 -0700 2006"
	ARCHIVE_TIME_FORMAT = "2006-01-02 15:04:05 -0700"
)

type Token struct {
	RequestToken string `json:"requestToken"`
	RequestTokenSecret string `json:"requestTokenSecret"`
	AccessToken string `json:"accessToken"`
	AccessTokenSecret string `json:"accessTokenSecret"`
	ConsumerKey string `json:"consumerKey"`
	ConsumerKeySecret string `json:"consumerKeySecret"`
	ScreenName string `json:"screenName"`
}

type Media struct {
	IdStr string
	Url string
	ExpandedUrl string
	Type string
	MediaUrl string
}

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
}

type SearchTerm struct {
	Original string
	Text string
	Upper string
	Quoted bool
	RegExp *regexp.Regexp
}

func init() {
	// default pages
	http.HandleFunc("/index.html", indexHandler)
	http.HandleFunc("/index", indexHandler)
	// handles all /.*
	http.HandleFunc("/", indexOrErrorHandler)
	// routes
	http.HandleFunc("/latest", indexHandler)
	http.HandleFunc("/best", indexHandler)
	http.HandleFunc("/search", indexHandler)
	http.HandleFunc("/error", indexHandler)

	// ajax calls
	http.HandleFunc("/user", userHandler)
	http.HandleFunc("/tweets", tweetsHandler)
	http.HandleFunc("/tweet", tweetHandler)

	// cron requests
	http.HandleFunc("/fetch", fetchTweetsHandler)
	http.HandleFunc("/update/tweets", updateTweetsHandler)
	http.HandleFunc("/update/user", updateUserHandler)

	// admin page requests
	http.HandleFunc("/admin", indexHandler)
	http.HandleFunc("/admin/archive/import", archiveImportHandler)
	http.HandleFunc("/admin/archive/export", archiveExportHandler)

	twitterApi = loadCredentials()
}

func archiveExportHandler(w http.ResponseWriter, r *http.Request) {
	//ctx := appengine.NewContext(r)
}

func archiveImportHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	reader := csv.NewReader(r.Body)
	// user, _ := getUser(ctx)
	// userIdStr := fmt.Sprintf("%v", user.Id)
	records, err := reader.ReadAll()
	if err != nil {
		log.Errorf(ctx, "Error parsing csv file: %v", err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}
	if len(records) > 0 {
		headers := records[0]
		rows := records[1:]
		log.Infof(ctx, "Import headers: %v", headers)
		log.Infof(ctx, "Import row count: %v", len(rows))
		var (
			//ids map[string]bool = make(map[string]bool)
			rowMaps []map[string]string = make([]map[string]string, len(rows))
		)
		for ind, row := range rows {
			for i, header := range headers {
				if rowMaps[ind] == nil {
					rowMaps[ind] = make(map[string]string)
				}
				rowMaps[ind][header] = row[i]
			}
			//ids[rowMaps[ind]["tweet_id"]] = false
		}
		//TODO: get all ids to match for reply_tos?
		tweets := []MyTweet{}
		for _, row := range rowMaps {
			// if row["retweeted_status_id"] != "" || (row["in_reply_to_status_id"] != "" &&
			// 	row["in_reply_to_user_id"] != userIdStr) {
			if row["retweeted_status_id"] != "" || row["in_reply_to_status_id"] != "" {
				//log.Debugf(ctx, "skipping row: %v", row)
			} else {
				id, _ := strconv.Atoi(row["tweet_id"])
				tweets = append(tweets, MyTweet{
					Id: int64(id),
					IdStr: row["tweet_id"],
					Created: makeTimestamp(parseTimestamp(row["timestamp"], ARCHIVE_TIME_FORMAT)),
					Updated: makeTimestamp(time.Now()),
					Url: TWITTER_URL + screenName + "/status/" + row["tweet_id"],
					Deleted: false,
					Media: nil,
				})
			}
		}

		log.Infof(ctx, "importable rows: %v", len(tweets))
		tweets, err = checkTweets(ctx, tweets)
		if err != nil {
			log.Errorf(ctx, "Error checking tweets from csv file: %v", err)
			http.Error(w, "Error", http.StatusInternalServerError)
			return
		}
		log.Infof(ctx, "checked tweets: %v. Storing...", len(tweets))

		err = storeTweets(ctx, tweets)
		if err != nil {
			log.Errorf(ctx, "Error storing tweets from csv file: %v", err)
			http.Error(w, "Error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func tweetHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	params := r.URL.Query()
	id, _ := strconv.Atoi(params.Get("id"))

	key := datastore.NewKey(ctx, "MyTweet", "", int64(id), nil)
	tweet := new(MyTweet)
	err := datastore.Get(ctx, key, tweet)
	if err != nil {
		log.Errorf(ctx, "Error getting tweet from datastore: %v", err)
		http.Error(w, "Error", http.StatusNotFound)
		return
	}

	var tweetJson []byte
	tweetJson, err = json.Marshal(tweet)
	if err != nil {
		log.Errorf(ctx, "Error marshaling json for tweet: %v", err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	w.Write(tweetJson)
}

func tweetsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	params := r.URL.Query()
	which := params.Get("type")
	var (
		tweets []MyTweet
		err error
	)
	if which == "best" {
		i, _ := strconv.Atoi(params.Get("page"))
		tweets, err = getBestTweets(ctx, max(1, i))
	} else if which == "latest" {
		i, _ := strconv.Atoi(params.Get("lastId"))
		tweets, err = getLatestTweets(ctx, i)
	} else if which == "search" {
		i, _ := strconv.Atoi(params.Get("page"))
		search := removePunctuation(strings.TrimSpace(params.Get("search")), false)
		tweets, err = getSearchTweets(ctx, search, params.Get("order"), i)
	} else {
		log.Errorf(ctx, "Error invalid tweet type: %v", which)
		http.Error(w, "Error", http.StatusNotFound)
		return
	}

	if err != nil {
		log.Errorf(ctx, "Error getting %v tweets: %v", which, err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	var tweetJson []byte
	tweetJson, err = json.Marshal(tweets)
	if err != nil {
		log.Errorf(ctx, "Error marshaling json for tweets: %v", err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	w.Write(tweetJson)
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	user, err := getUser(ctx)

	if err != nil {
		log.Errorf(ctx, "Error getting user: %v", err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	var userJson []byte
	userJson, err = json.Marshal(user)
	if err != nil {
		log.Errorf(ctx, "Error marshaling json for user: %v", err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	w.Write(userJson)
}

func indexOrErrorHandler(w http.ResponseWriter, r *http.Request) {
	name := path.Clean(r.URL.Path)
	ctx := appengine.NewContext(r)
	if r.Method == "GET" {
		if name == "/" {
			indexHandler(w, r)
		} else if name == "/favicon.ico" {
			w.WriteHeader(http.StatusNotFound)
		} else {
			http.Redirect(w, r, "/error", http.StatusMovedPermanently)
		}
	} else {
		log.Errorf(ctx, "Error wrong method: %v", r.Method)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// http.ServeFile(w, r, "html/admin.html")
	ctx := appengine.NewContext(r)
	user, err := getUser(ctx)
	if err != nil {
		log.Errorf(ctx, "Error fetching user: %v", err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	page := "html/main.html"
	name := path.Clean(r.URL.Path)
	if name == "/admin" {
		page = "html/admin.html"
	}

	var temp *template.Template
	temp, err = template.ParseFiles(page)

	if err != nil {
		log.Errorf(ctx, "Error parsing template: %v", err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	temp.Execute(w, user)
}

func updateUserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	isCron := r.Header.Get("X-Appengine-Cron")

	if isCron == "true" {
		_, err := fetchAndStoreUser(ctx)
		if err != nil {
			log.Errorf(ctx, "Error fetching and storing user", err)
			http.Error(w, "Error", http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	} else {
		log.Warningf(ctx, "unauthorized attempt to access cron /update/user")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}

func updateTweetsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	isCron := r.Header.Get("X-Appengine-Cron")

	if isCron == "true" {
		if err := updateDatastoreTweets(ctx); err != nil {
			log.Errorf(ctx, "Error updating db tweets: %v", err)
			http.Error(w, "Error", http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	} else {
		log.Warningf(ctx, "unauthorized attempt to access cron /update/tweets")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}

func fetchTweetsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	isCron := r.Header.Get("X-Appengine-Cron")

	if isCron == "true" {
		_, err := fetchAndStoreTweets(ctx)
		if err != nil {
			log.Errorf(ctx, "Error fetching and storing tweets", err)
			http.Error(w, "Error", http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	} else {
		log.Warningf(ctx, "unauthorized attempt to access cron /fetch")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}

func getUser(ctx context.Context) (*User, error) {
	var cached *User
	_, err := memcache.JSON.Get(ctx, MEMCACHE_USER_KEY + screenName, &cached)
	if err != nil {
		log.Errorf(ctx, "failed to fetch user from memcache: %v", err)
	}

	if cached == nil {
		var user *User
		user, err = getDataStoreUser(ctx, screenName)
		if err != nil || user == nil {
			fetchAndStoreUser(ctx)
			return user, nil
		}
		return user, nil
	}
	return cached, nil
}

func fetchAndStoreUser(ctx context.Context) (*User, error) {
	twitterApi.HttpClient.Transport = &urlfetch.Transport{Context: ctx}
	anacondaUser, err := twitterApi.GetUsersShow(screenName, url.Values{
		"include_entities": {"1"},
	})

	if err != nil {
		log.Errorf(ctx, "Error getting twitter user: %v", err)
		return nil, err
	}

	user := &User{
		ScreenName: anacondaUser.ScreenName,
		Id: anacondaUser.Id,
		Name: anacondaUser.Name,
		ProfileImageUrlHttps: anacondaUser.ProfileImageUrlHttps,
		Url: TWITTER_URL + anacondaUser.ScreenName,
		Updated: makeTimestamp(time.Now()),
		Description: anacondaUser.Description,
		Followers: anacondaUser.FollowersCount,
		Following: anacondaUser.FriendsCount,
		TweetCount: anacondaUser.StatusesCount,
		Location: anacondaUser.Location,
		Verified: anacondaUser.Verified,
		Link: anacondaUser.URL,
	}

	store := &memcache.Item{
		Key: MEMCACHE_USER_KEY + screenName,
		Object: *user,
	}
	memcache.JSON.Set(ctx, store)

	if err = storeUser(ctx, user); err != nil {
		log.Errorf(ctx, "failed to store user: %v", err)
		return nil, err
	}

	return user, nil
}

func getSearchTweets(ctx context.Context, search string, order string, page int) ([]MyTweet, error) {
	var (
		tweets []MyTweet
		err error
	)

	if len(search) > MIN_SEARCH_LENGTH {
		terms := getTerms(search)

		if len(terms) > 0 {
			query := datastore.NewQuery("MyTweet").Filter("Deleted =", false).Order(order)
			tweets = []MyTweet{}
			_, err = query.GetAll(ctx, &tweets)
			if err != nil {
				log.Errorf(ctx, "Error getting best tweets from datastore: %v", err)
				return nil, err
			}

			tweets = searchTweets(tweets, terms)
			return getPage(tweets, page), nil
		}
	}

	return tweets, nil
}

func getLatestTweets(ctx context.Context, lastId int) ([]MyTweet, error) {
	var (
		tweets []MyTweet
		err error
	)

	query := datastore.NewQuery("MyTweet").
		Filter("Deleted =", false).
		Limit(TWEETS_TO_FETCH).
		Order("-Id")
	if lastId != 0 {
		query = query.Filter("Id <", lastId)
	}

	tweets = []MyTweet{}
	_, err = query.GetAll(ctx, &tweets)
	if err != nil {
		log.Errorf(ctx, "Error getting latest tweets from datastore: %v", err)
		return nil, err
	}

	return tweets, nil
}

func getBestTweets(ctx context.Context, page int) ([]MyTweet, error) {
	var (
		tweets []MyTweet
		err error
	)

	query := datastore.NewQuery("MyTweet").
		Filter("Deleted =", false).
		Order("-Faves").
		Order("-Rts").
		Order("-Ratio")
	tweets = []MyTweet{}
	_, err = query.GetAll(ctx, &tweets)
	if err != nil {
		log.Errorf(ctx, "Error getting best tweets from datastore: %v", err)
		return nil, err
	}


	return getPage(tweets, page), err
}

func getPage(tweets []MyTweet, page int) []MyTweet {
	length := len(tweets)
	if length > 0 {
		if page <= 1 {
			return tweets[0 : min(TWEETS_TO_FETCH, length)]
		}
		if page * TWEETS_TO_FETCH < length {
			return tweets[(page - 1) * TWEETS_TO_FETCH : min(page * TWEETS_TO_FETCH, length)]
		}
		return nil
	}
	return tweets
}

func fetchAndStoreTweets(ctx context.Context) ([]MyTweet, error) {
	var tweets []MyTweet

	lastTweet, _ := getLatestTweet(ctx)
	lastTweetID := int64(0)
	if lastTweet != nil {
		lastTweetID = lastTweet.Id
	}

	tweets, err := fetchTweets(ctx, tweets, int64(0), lastTweetID)
	if err != nil {
		log.Errorf(ctx, "error fetching tweets: %v", err)
		return nil, err
	}

	if err = storeTweets(ctx, tweets); err != nil {
		return nil, err
	}
	return tweets, nil
}

func updateDatastoreTweets(ctx context.Context) (err error) {
	tweets := []MyTweet{}
	query := datastore.NewQuery("MyTweet").Order("Updated").Filter("Deleted =", false)
	_, err = query.GetAll(ctx, &tweets)

	if err != nil {
		log.Errorf(ctx, "error getting tweets: %v", err)
		return err
	}

	log.Infof(ctx, "Checking tweets: %v", len(tweets))
	tweets, err = checkTweets(ctx, tweets)
	if err != nil {
		return err
	}
	// Iterate over tweets and fetch from Twitter
	// Update values
	// Store
	log.Infof(ctx, "Looked up tweets: %v", len(tweets))

	return storeTweets(ctx, tweets)
}

func checkTweets(ctx context.Context, tweets []MyTweet) ([]MyTweet, error) {
	if len(tweets) == 0 {
		return nil, nil
	}
	twitterApi.HttpClient.Transport = &urlfetch.Transport{Context: ctx}

	out := []MyTweet{}
	ids := []int64{}
	vals := url.Values{
		"trim_user": {"1"},
		"include_entities": {"1"},
	}

	toCheck := tweets[: min(MAX_API_LOOKUP_SIZE, len(tweets))]
	rest := tweets[min(MAX_API_LOOKUP_SIZE, len(tweets)) :]

	for _, t := range toCheck {
		ids = append(ids, t.Id)
	}

	aTweets, err := twitterApi.GetTweetsLookupByIds(ids, vals)
	if err != nil {
		return nil, err
	}

	//process aTweets
	for _, t := range toCheck {
		var (
			aTweet anaconda.Tweet
			found bool = false
		)
		for _, aTweet = range aTweets {
			if aTweet.Id == t.Id {
				found = true
				break
			}
		}

		if found == false {
			log.Infof(ctx, "Tweet deleted: %v", t.Id)
			t.Deleted = true
		} else {
			t.Faves = aTweet.FavoriteCount
			t.Rts = aTweet.RetweetCount
			t.Ratio = getRatio(aTweet.FavoriteCount, aTweet.RetweetCount)
			if t.Media == nil || len(t.Media) == 0 {
				t.Media = getMedia(&aTweet)
			}
			if t.Text == "" {
				t.Text = aTweet.FullText
			}
		}
		t.Updated = makeTimestamp(time.Now())

		if testTweet(aTweet) {
			out = append(out, t)
		}
	}

	if len(rest) > 0 {
		t, e := checkTweets(ctx, rest)
		if e != nil {
			return nil, e
		} else {
			out = append(out, t...)
		}
	}

	return out, nil
}

func getLatestTweet(ctx context.Context) (*MyTweet, error) {
	var tweets []MyTweet = []MyTweet{}
	q := datastore.NewQuery("MyTweet").Limit(1).Order("-Id")
	_, err := q.GetAll(ctx, &tweets)
	if err != nil || len(tweets) == 0 {
		log.Errorf(ctx, "error getting last stored tweet: %v", err)
		return nil, err
	}
	return &tweets[0], nil
}

func storeTweets(ctx context.Context, tweets []MyTweet) error {
	keys := []*datastore.Key{}
	for _, tweet := range tweets {
		key := datastore.NewKey(ctx, "MyTweet", "", tweet.Id, nil)
		keys = append(keys, key)
	}

	length := len(keys)
	for i := 0; i < length; i += MAX_PUT_SIZE {
		max := min(i + MAX_PUT_SIZE, length)
		slicedKeys := keys[i:max]
		slicedTweets := tweets[i:max]
		newKeys, err := datastore.PutMulti(ctx, slicedKeys, slicedTweets)
		if err != nil {
			log.Errorf(ctx, "Error storing tweets in db: %v", err)
			return err
		} else {
			log.Infof(ctx, "Saved tweets: %v", len(newKeys))
		}
	}
	return nil
}

func storeUser(ctx context.Context, user *User) error {
	key := datastore.NewKey(ctx, "User", user.ScreenName, 0, nil)
	newKey, err := datastore.Put(ctx, key, user)
	if err != nil {
		log.Errorf(ctx, "Error storing user in db: %v", err)
		return err
	} else {
		log.Infof(ctx, "Stored user: %v", newKey)
	}
	return nil
}

func getDataStoreUser(ctx context.Context, screenName string) (*User, error) {
	key := datastore.NewKey(ctx, "User", screenName, 0, nil)
	user := new(User)
	err := datastore.Get(ctx, key, user)
	if err != nil {
		log.Errorf(ctx, "Error getting user from datastore: %v", err)
		return nil, err
	}
	return user, nil
}

func fetchTweets(ctx context.Context, tweets []MyTweet, lastId int64, latestId int64) ([]MyTweet, error) {
	twitterApi.HttpClient.Transport = &urlfetch.Transport{Context: ctx}
	log.Infof(ctx, "Fetching Tweets (lastId): %v, (latestId): %v", lastId, latestId)
	vals := url.Values{
		"screen_name": {screenName},
		"count": {"200"},
		"trim_user": {"1"},
		"exclude_replies": {"1"},
		"include_rts": {"0"},
	}
	if lastId > 0 {
		vals.Add("max_id", fmt.Sprintf("%v", lastId - 1))
	}
	if latestId > 0 {
		vals.Add("since_id", fmt.Sprintf("%v", latestId))
	}
	aTweets, err := twitterApi.GetUserTimeline(vals)
	if err != nil {
		return tweets, err
	}

	// return after fetching all the tweets we can
	if len(aTweets) == 0 {
		return tweets, nil
	}

	procTweets, newLastId := processTweets(aTweets)
	tweets = append(tweets, procTweets...)

	log.Infof(ctx, "Fetched Tweets: %v; (newLastId): %v", len(tweets), newLastId)

	return fetchTweets(ctx, tweets, newLastId, latestId)
}

func processTweets(tweets []anaconda.Tweet) ([]MyTweet, int64) {
	out := []MyTweet{}
	lastId := int64(0)
	for _, tweet := range tweets {
		if lastId == 0 || tweet.Id < lastId {
			lastId = tweet.Id
		}
		if testTweet(tweet) {
			myTweet := MyTweet{
				Ratio: getRatio(tweet.FavoriteCount, tweet.RetweetCount),
				IdStr: tweet.IdStr,
				Faves: tweet.FavoriteCount,
				Rts: tweet.RetweetCount,
				Id: tweet.Id,
				Created: makeTimestamp(parseTimestamp(tweet.CreatedAt, SEARCH_TIME_FORMAT)),
				Updated: makeTimestamp(time.Now()),
				Text: tweet.FullText,
				Url: TWITTER_URL + screenName + "/status/" + tweet.IdStr,
				Deleted: false,
				Media: getMedia(&tweet),
			}

			out = append(out, myTweet)
		}
	}

	return out, lastId
}

func getMedia(tweet *anaconda.Tweet) (media []Media) {
	if len(tweet.Entities.Media) > 0 {
		//Has media entities...
		for _, ent := range tweet.Entities.Media {
			media = append(media, Media{
				Type: ent.Type,
				IdStr: ent.Id_str,
				Url: ent.Url,
				ExpandedUrl: ent.Expanded_url,
				MediaUrl: ent.Media_url_https,
			})
		}
	}
	return media
}

func getTerms(search string) (terms [][]SearchTerm) {
	// ORs[] of ANDs[]
	ors := strings.Split(search, " OR ")
	for _, or := range ors {
		split := strings.Fields(or)
		o := []SearchTerm{}
		for _, term := range split {
			if len(term) > MIN_SEARCH_LENGTH {
				// make all uppercase
				var (
					quoted bool = false
					reg *regexp.Regexp = nil
					str string = term
				)
				if strings.HasPrefix(term, "\"") == true && strings.HasSuffix(term, "\"") == true {
					quoted = true
					runes := []rune(term)
					str = string(runes[1 : len(term) - 1])
					reg, _ = regexp.Compile("(^| )" + strings.ToUpper(str) + "( |$)")
				}
				o = append(o, SearchTerm{
					Original: term,
					Text: str,
					Upper: strings.ToUpper(str),
					Quoted: quoted,
					RegExp: reg,
				})
			}
		}

		if len(o) > 0 {
			terms = append(terms, o)
		}
	}
	return terms
}

func searchTweets(tweets []MyTweet, terms [][]SearchTerm) (ret []MyTweet) {
	for _, tweet := range tweets {
		if tweetMatchesTerms(tweet, terms) {
			ret = append(ret, tweet)
		}
	}
	return ret
}

func removePunctuation(text string, quotes bool) string {
	var reg *regexp.Regexp
	if quotes == true {
		reg, _ = regexp.Compile("[^a-zA-Z0-9 ]")
	} else {
		reg, _ = regexp.Compile("[^a-zA-Z0-9 \"]")
	}
	return reg.ReplaceAllString(text, "")
}

func tweetMatchesTerms(tweet MyTweet, terms [][]SearchTerm) bool {
	text := strings.ToUpper(removePunctuation(tweet.Text, true))
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

func testTweet(tweet anaconda.Tweet) bool {
	return tweet.Id != 0 &&
		len(tweet.Entities.User_mentions) <= 0 &&
		len(tweet.Entities.Urls) <= 0
}

func getRatio(favs int, rts int) float32 {
	if favs <= 0 {
		return float32(0)
	}
	return float32(rts) / float32(favs)
}

func parseTimestamp(str string, format string) time.Time {
	created, err := time.Parse(format, str)
	if err != nil {
		created = time.Time{}
	}
	return created
}

func makeTimestamp(t time.Time) int64 {
	return t.Unix()
}

func min(num1 int, num2 int) int {
	return int(math.Min(float64(num1), float64(num2)))
}

func max(num1 int, num2 int) int {
	return int(math.Max(float64(num1), float64(num2)))
}

func loadCredentials() (api *anaconda.TwitterApi) {
	credentials, err := ioutil.ReadFile("credentials")
	if err != nil {
		os.Exit(1)
	}

	var token Token
	err = json.Unmarshal([]byte(credentials), &token)
	if err != nil {
		os.Exit(1)
	}

	screenName = token.ScreenName

	anaconda.SetConsumerKey(token.ConsumerKey)
	anaconda.SetConsumerSecret(token.ConsumerKeySecret)
	//api = anaconda.NewTwitterApi(token.AccessToken, token.AccessTokenSecret)
	api = anaconda.NewTwitterApi("", "")

	return api
}
