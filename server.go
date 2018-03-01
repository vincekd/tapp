package tapp

import (
	"fmt"
	"time"
	"math"
	"strings"
	"regexp"
	"context"
	"path"
	"bytes"
	//"errors"
	"net/http"
	"net/url"
	"strconv"
	"html/template"
	"encoding/json"
	"encoding/csv"
	"encoding/xml"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/memcache"
	"google.golang.org/appengine/datastore"
	"github.com/ChimeraCoder/anaconda"
)

var (
	TwitterApi *anaconda.TwitterApi
	MyToken Credentials
)

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
	http.HandleFunc("/tweet", tweetHandler)
	http.HandleFunc("/tweets/latest", tweetsHandler)
	http.HandleFunc("/tweets/best", tweetsHandler)
	http.HandleFunc("/tweets/search", tweetsHandler)

	// cron requests
	http.HandleFunc("/fetch", fetchTweetsHandler)
	http.HandleFunc("/update/tweets", updateTweetsHandler)
	http.HandleFunc("/update/user", updateUserHandler)

	// admin page requests
	http.HandleFunc("/admin", indexHandler)
	http.HandleFunc("/admin/archive/import", archiveImportHandler)

	// rss feed
	http.HandleFunc("/xml/rss/tweets/latest.xml", rssFeedHandler)
	http.HandleFunc("/xml/rss/tweets/best.xml", rssFeedHandler)

	TwitterApi, MyToken = LoadCredentials()
}

func rssFeedHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	var (
		tweets []MyTweet
		err error
	)
	which := strings.Replace(path.Clean(r.URL.Path), "/xml/rss/tweets/", "", 1)
	if which == "latest.xml" {
		tweets, err = getLatestTweets(ctx, 0)
	} else if which == "best.xml" {
		tweets, err = getBestTweets(ctx, 0)
	} else {
		log.Errorf(ctx, "Rss feed: %v not found: %v", which, err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	if err != nil {
		log.Errorf(ctx, "Error getting latest tweets: %v", err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	type XmlTweet struct {
		XMLName xml.Name `xml:"item"`
		Title string `xml:"title"`
		Link string `xml:"link"`
		Guid string `xml:"guid"`
		PubDate string `xml:"pubDate"`
		Description string `xml:"description"`
	}
	type Image struct {
		XMLName xml.Name `xml:"image"`
		Title string `xml:"title"`
		Url string `xml:"url"`
		Link string `xml:"link"`
	}
	type Channel struct {
		XMLName xml.Name `xml:"channel"`
		Title string `xml:"title"`
		Link string `xml:"link"`
		Language string `xml:"language"`
		Copyright string `xml:"copyright"`
		Image Image
		XmlTweets []XmlTweet
	}
	type Headers struct {
		XMLName xml.Name `xml:"rss"`
		Version string `xml:"version,attr"`
		Channel Channel
	}

	xmlTweets := make([]XmlTweet, len(tweets))
	for i, tweet := range tweets {
		t := time.Unix(tweet.Created, 0).Format(XML_RSS_TIME_FORMAT)
		xmlTweets[i] = XmlTweet{
			Title: t + " Tweet",
			Link: tweet.Url,
			Guid: tweet.Url,
			PubDate: t,
			Description: tweet.Text,
		}
	}

	now := time.Now().Format("2006")
	user, _ := getUser(ctx)
	url := r.URL.String()
	buf := &bytes.Buffer{}
	encoder := xml.NewEncoder(buf)

	encoder.Indent("", "  ")
	encoder.Encode(Headers{
		Version: "2.0",
		Channel: Channel{
			XmlTweets: xmlTweets,
			Title: "@" + user.ScreenName + " Latest Tweets",
			Link: url,
			Language: "en-US",
			Copyright: "Copyright " + now + " @" + user.ScreenName,
			Image: Image{
				Title: user.ScreenName + " Tweets",
				Url: user.ProfileImageUrlHttps,
				Link: url,
			},
		},
	})
	encoder.Flush()

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n"))
	w.Write(bytes.Replace(buf.Bytes(), []byte("&#xA;"), []byte("\n"), -1))
}

func archiveImportHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	reader := csv.NewReader(r.Body)

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

			} else {
				id, _ := strconv.Atoi(row["tweet_id"])
				tweets = append(tweets, MyTweet{
					Id: int64(id),
					IdStr: row["tweet_id"],
					Created: parseTimestamp(row["timestamp"], ARCHIVE_TIME_FORMAT).Unix(),
					Updated: time.Now().Unix(),
					Url: TWITTER_URL + MyToken.ScreenName + "/status/" + row["tweet_id"],
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

	tweet := MyTweet{Id: int64(id)}
	err := datastore.Get(ctx, tweet.GetKey(ctx), &tweet)
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
	var (
		tweets []MyTweet
		err error
	)

	i, _ := strconv.Atoi(params.Get("page"))
	which := strings.Replace(path.Clean(r.URL.Path), "/tweets/", "", 1)
	switch which {
	case "best":
		tweets, err = getBestTweets(ctx, i)
	case "latest":
		tweets, err = getLatestTweets(ctx, i)
	case "search":
		tweets, err = getSearchTweets(ctx, i, params.Get("search"), params.Get("order"))
	default:
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
		reg, _ := regexp.Compile("/tweet/[0-9]+")
		if name == "/" || reg.MatchString(name) {
			indexHandler(w, r)
		} else if name == "/favicon.ico" {
			w.WriteHeader(http.StatusNotFound)
		} else {
			//TODO: make /error/404 or something...
			http.Redirect(w, r, "/error", http.StatusMovedPermanently)
		}
	} else {
		log.Errorf(ctx, "Error wrong method: %v", r.Method)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
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
	mainPage := struct {
		User *User
		GaKey string
		HasGaKey bool
	} {
		User: user,
		GaKey: MyToken.GaKey,
		// disable if localhost or no ga key supplied in credentials
		HasGaKey: MyToken.GaKey != "" && isLocalhost(r.RemoteAddr) == false,
	}

	if err != nil {
		log.Errorf(ctx, "Error parsing template: %v", err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	err = temp.Execute(w, mainPage)
	if err != nil {
		log.Errorf(ctx, "Error executing template: %v", err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}
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
	_, err := memcache.JSON.Get(ctx, MEMCACHE_USER_KEY + MyToken.ScreenName, &cached)
	if err != nil {
		log.Errorf(ctx, "failed to fetch user from memcache: %v", err)
	}

	if cached == nil {
		var user *User
		user, err = getDataStoreUser(ctx, MyToken.ScreenName)
		if err != nil || user == nil {
			fetchAndStoreUser(ctx)
			return user, nil
		}
		return user, nil
	}
	return cached, nil
}

func fetchAndStoreUser(ctx context.Context) (*User, error) {
	TwitterApi.HttpClient.Transport = &urlfetch.Transport{Context: ctx}
	anacondaUser, err := TwitterApi.GetUsersShow(MyToken.ScreenName, url.Values{
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
		Updated: time.Now().Unix(),
		Description: anacondaUser.Description,
		Followers: anacondaUser.FollowersCount,
		Following: anacondaUser.FriendsCount,
		TweetCount: anacondaUser.StatusesCount,
		Location: anacondaUser.Location,
		Verified: anacondaUser.Verified,
		Link: anacondaUser.URL,
	}

	store := &memcache.Item{
		Key: MEMCACHE_USER_KEY + MyToken.ScreenName,
		Object: *user,
	}
	memcache.JSON.Set(ctx, store)

	if err = user.Store(ctx); err != nil {
		log.Errorf(ctx, "failed to store user: %v", err)
		return nil, err
	}

	return user, nil
}

func getSearchTweets(ctx context.Context, page int, search string, order string) ([]MyTweet, error) {
	var (
		tweets []MyTweet
		err error
	)

	search = RemovePunctuation(strings.TrimSpace(search), false)
	if len(search) > MIN_SEARCH_LENGTH {
		terms := getTerms(search)
		//log.Debugf(ctx, "terms: %v", terms)

		if len(terms) > 0 {
			query := datastore.NewQuery("MyTweet").
				Filter("Deleted =", false).
				Order(order).
				Offset(page * TWEETS_TO_FETCH)
			tweets = []MyTweet{}
			_, err = query.GetAll(ctx, &tweets)
			if err != nil {
				log.Errorf(ctx, "Error getting best tweets from datastore: %v", err)
				return nil, err
			}

			tweets = searchTweets(tweets, terms)

			length := len(tweets)
			if length > 0 {
				num := page * TWEETS_TO_FETCH
				if num < length {
					return tweets[num : min(num + TWEETS_TO_FETCH, length)], nil
				}
				return nil, nil
			}
			return tweets, nil
		}
	}

	return tweets, nil
}

func getLatestTweets(ctx context.Context, page int) ([]MyTweet, error) {
	var (
		tweets []MyTweet
		err error
	)

	query := datastore.NewQuery("MyTweet").
		Filter("Deleted =", false).
		Order("-Id").
		Limit(TWEETS_TO_FETCH).
		Offset(page * TWEETS_TO_FETCH)

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
		Order("-Ratio").
		Limit(TWEETS_TO_FETCH).
		Offset(page * TWEETS_TO_FETCH)

	tweets = []MyTweet{}
	_, err = query.GetAll(ctx, &tweets)
	if err != nil {
		log.Errorf(ctx, "Error getting best tweets from datastore: %v", err)
		return nil, err
	}

	return tweets, nil
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
	// Iterate over tweets and fetch from Twitter
	// Update values
	// Store
	tweets, err = checkTweets(ctx, tweets)
	if err != nil {
		return err
	}
	log.Infof(ctx, "Looked up tweets: %v", len(tweets))

	return storeTweets(ctx, tweets)
}

func checkTweets(ctx context.Context, tweets []MyTweet) ([]MyTweet, error) {
	if len(tweets) == 0 {
		return nil, nil
	}
	TwitterApi.HttpClient.Transport = &urlfetch.Transport{Context: ctx}

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

	aTweets, err := TwitterApi.GetTweetsLookupByIds(ids, vals)
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
		t.Updated = time.Now().Unix()

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
		keys = append(keys, tweet.GetKey(ctx))
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

func getDataStoreUser(ctx context.Context, screenName string) (*User, error) {
	user := User{ScreenName: screenName}
	err := datastore.Get(ctx, user.GetKey(ctx), &user)
	if err != nil {
		log.Errorf(ctx, "Error getting user from datastore: %v", err)
		return nil, err
	}
	return &user, nil
}

func fetchTweets(ctx context.Context, tweets []MyTweet, lastId int64, latestId int64) ([]MyTweet, error) {
	TwitterApi.HttpClient.Transport = &urlfetch.Transport{Context: ctx}
	log.Infof(ctx, "Fetching Tweets (lastId): %v, (latestId): %v", lastId, latestId)
	vals := url.Values{
		"screen_name": {MyToken.ScreenName},
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
	aTweets, err := TwitterApi.GetUserTimeline(vals)
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
				Created: parseTimestamp(tweet.CreatedAt, SEARCH_TIME_FORMAT).Unix(),
				Updated: time.Now().Unix(),
				Text: tweet.FullText,
				Url: TWITTER_URL + MyToken.ScreenName + "/status/" + tweet.IdStr,
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
	quoted := []string{}
	reg, _ := regexp.Compile("\"[^\"]*\"")
	//TODO: unpaired quotes
	newStr := reg.ReplaceAllStringFunc(search, func(quote string) string {
		quote = strings.Trim(quote, "\"")
		if len(quote) > MIN_SEARCH_LENGTH {
			quoted = append(quoted, quote)
			return "$$"
		}
		return ""
	})
	ors := strings.Split(newStr, " OR ")
	for _, or := range ors {
		split := strings.Fields(or)
		o := []SearchTerm{}
		for _, term := range split {
			if term == "$$" {
				// unshift from quoted slice
				str := quoted[0]
				quoted = quoted[1:]
				reg, _ = regexp.Compile("(^| )" + strings.ToUpper(str) + "( |$)")
				o = append(o, SearchTerm{
					Text: str,
					Upper: strings.ToUpper(str),
					Quoted: true,
					RegExp: reg,
				})
			} else if len(term) > MIN_SEARCH_LENGTH {
				term = RemovePunctuation(term, true)
				o = append(o, SearchTerm{
					Text: term,
					Upper: strings.ToUpper(term),
					Quoted: false,
					RegExp: nil,
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
		if tweet.MatchesTerms(terms) {
			ret = append(ret, tweet)
		}
	}
	return ret
}

func RemovePunctuation(text string, quotes bool) string {
	var reg *regexp.Regexp
	// other replacements?
	text = strings.Replace(text, "'", "", -1)
	text = strings.Replace(text, "&", " and ", -1)
	text = strings.Replace(text, "%", " percent ", -1)
	if quotes == true {
		reg, _ = regexp.Compile("[^a-zA-Z0-9#@ ]")
	} else {
		reg, _ = regexp.Compile("[^a-zA-Z0-9#@ \"]")
	}

	text = reg.ReplaceAllString(text, " ")
	reg, _ = regexp.Compile(" +")
	return reg.ReplaceAllString(text, " ")
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
	stamp, err := time.Parse(format, str)
	if err != nil {
		stamp = time.Time{}
	}
	return stamp
}

func min(num1 int, num2 int) int {
	return int(math.Min(float64(num1), float64(num2)))
}

func max(num1 int, num2 int) int {
	return int(math.Max(float64(num1), float64(num2)))
}

func isLocalhost(addr string) bool {
	return addr == "127.0.0.1" || addr == "::1"
}
