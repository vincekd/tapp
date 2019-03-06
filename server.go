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
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"html/template"
	"encoding/json"
	"encoding/csv"
	"encoding/xml"
	"archive/zip"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/memcache"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/file"
	"github.com/ChimeraCoder/anaconda"
	"cloud.google.com/go/storage"
)

var (
	TwitterApi *anaconda.TwitterApi
	MyToken Credentials
)

type appEngineHandler func(context.Context, http.ResponseWriter, *http.Request) error

func appHandler(handler appEngineHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)
		if err := handler(ctx, w, r); err != nil {
			log.Errorf(ctx, "Handler error: %v", err)
			http.Error(w, "Error", http.StatusInternalServerError)
		}
	}
}

func validateCron(handler appEngineHandler) appEngineHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		isCron := r.Header.Get("X-Appengine-Cron")
		if isCron != "true" {
			log.Warningf(ctx, "unauthorized attempt to access cron /unretweet")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return nil
		}
		return handler(ctx, w, r)
	}
}

func init() {
	// default pages
	http.HandleFunc("/index.html", appHandler(indexHandler))
	http.HandleFunc("/index", appHandler(indexHandler))
	// handles all /.*
	http.HandleFunc("/", appHandler(indexOrErrorHandler))
	// routes
	http.HandleFunc("/latest", appHandler(indexHandler))
	http.HandleFunc("/best", appHandler(indexHandler))
	http.HandleFunc("/search", appHandler(indexHandler))
	http.HandleFunc("/error", appHandler(indexHandler))

	// ajax calls
	http.HandleFunc("/user", appHandler(userHandler))
	http.HandleFunc("/tweet", appHandler(tweetHandler))
	http.HandleFunc("/tweets/latest", appHandler(tweetsHandler))
	http.HandleFunc("/tweets/best", appHandler(tweetsHandler))
	http.HandleFunc("/tweets/search", appHandler(tweetsHandler))

	// cron requests
	http.HandleFunc("/fetch", appHandler(validateCron(fetchTweetsHandler)))
	http.HandleFunc("/update/tweets", appHandler(validateCron(updateTweetsHandler)))
	http.HandleFunc("/update/user", appHandler(validateCron(updateUserHandler)))
	http.HandleFunc("/unretweet", appHandler(validateCron(unretweetHanlder)))

	// admin page requests
	http.HandleFunc("/admin", appHandler(indexHandler))
	http.HandleFunc("/admin/archive/import", appHandler(archiveImportHandler))
	http.HandleFunc("/admin/archive/export", appHandler(archiveExportHandler))
	http.HandleFunc("/admin/delete", appHandler(toggleDeletedHandler))

	// media
	http.HandleFunc("/media", appHandler(mediaHandler))

	// rss feed
	http.HandleFunc("/feed/latest.xml", appHandler(feedHandler))

	TwitterApi, MyToken = LoadCredentials(false)
}

func mediaHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	filePath := r.URL.Query().Get("file")
	if filePath == "" {
		return fmt.Errorf("No file path passed")
	}

	bucket, err := getBucket(ctx)
	if err != nil {
		return fmt.Errorf("Error getting bucket: %v", err)
	}

	rc, err := bucket.Object(filePath).NewReader(ctx)
	if err != nil {
		return fmt.Errorf("readFile: unable to open file %q: %v", filePath, err)
	}

	defer rc.Close()
	slurp, err := ioutil.ReadAll(rc)
	if err != nil {
		return fmt.Errorf("readFile: unable to read data from file %q: %v", filePath, err)
	}

	_, err = w.Write(slurp)
	return err
}

func toggleDeletedHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	params := r.URL.Query()
	id, _ := strconv.Atoi(params.Get("id"))
	tweet := MyTweet{Id: int64(id)}
	log.Infof(ctx, "deleting tweet: %+v", tweet)

	if err := datastore.Get(ctx, tweet.GetKey(ctx), &tweet); err != nil {
		//TODO: return statusnotfound
		return fmt.Errorf("Error getting tweet from datastore: %v", err)
	}
	tweet.Deleted = !tweet.Deleted

	_, err := datastore.Put(ctx, tweet.GetKey(ctx), &tweet)
	return err
}

func feedHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	tweets, err := getLatestTweets(ctx, 0)
	if err != nil {
		return fmt.Errorf("Error getting latest tweets: %v", err)
	}

	type Link struct {
		Href string `xml:"href,attr"`
		Type string `xml:"rel,attr"`
	}
	type XmlTweet struct {
		XMLName xml.Name `xml:"entry"`
		Title string `xml:"title"`
		Link Link `xml:"link"`
		Id string `xml:"id"`
		Updated string `xml:"updated"`
		Summary string `xml:"summary"`
		Content string `xml:"content"`
		Author string `xml:"author>name"`
	}

	var user *User
	var last *MyTweet
	user, err = getUser(ctx)
	if err != nil {
		return fmt.Errorf("Error getting user: %v", err)
	}
	last, err = getLatestTweet(ctx)
	if err != nil {
		return fmt.Errorf("Error getting latest tweet: %v", err)
	}

	xmlTweets := make([]XmlTweet, len(tweets))
	for i, tweet := range tweets {
		t := time.Unix(tweet.Created, 0)
		xmlTweets[i] = XmlTweet{
			Title: t.Format(FEED_HEADER_FORMAT) + " Tweet",
			Link: Link{tweet.Url, "alternate"},
			Id: tweet.IdStr,
			Updated: t.Format(XML_ATOM_TIME_FORMAT),
			Summary: tweet.Text[:min(len(tweet.Text), SUMMARY_LENGTH)],
			//Content: "<![CDATA[ " + strings.Replace(tweet.Text, "\n", "\n<br/>", -1) + " ]]>",
			Content: "<![CDATA[ " + tweet.Text + " ]]>",
			Author: "@" + user.ScreenName,
		}
	}

	url := r.URL.String()
	buf := &bytes.Buffer{}
	year := time.Now().Format("2006")
	encoder := xml.NewEncoder(buf)

	encoder.Indent("", "  ")
	encoder.Encode(struct {
		XMLName xml.Name `xml:"feed"`
		Xmlns string `xml:"xmlns,attr"`
		Title string `xml:"title"`
		SubTitle string `xml:"subtitle"`
		Link Link `xml:"link"`
		Updated string `xml:"updated"`
		Id string `xml:"id"`
		Icon string `xml:"icon"`
		Logo string `xml:"logo"`
		Rights string `xml:"rights"`
		XmlTweets []XmlTweet
	} {
		Xmlns: "http://www.w3.org/2005/Atom",
		Title: "@" + user.ScreenName + " Latest Tweets Feed",
		Link: Link{url, "self"},
		Updated: time.Unix(last.Updated, 0).Format(XML_ATOM_TIME_FORMAT),
		Id: url,
		Icon: user.ProfileImageUrlHttps,
		Logo: user.ProfileImageUrlHttps,
		Rights: "© " + year + " "+ user.ScreenName,
		XmlTweets: xmlTweets,
	})
	encoder.Flush()

	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n"))
	var b []byte = bytes.Replace(buf.Bytes(), []byte("&#xA;"), []byte("<br/>\n"), -1)
	b = bytes.Replace(b, []byte("&lt;"), []byte("<"), -1)
	b = bytes.Replace(b, []byte("&gt;"), []byte(">"), -1)
	//w.Write(bytes.Replace(buf.Bytes(), []byte("&#xA;"), []byte("<br/>\n"), -1))
	//w.Write(buf.Bytes())
	_, err = w.Write(b)
	return err
}

func archiveExportHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	query := datastore.NewQuery("MyTweet").Order("Created")
	tweets := []MyTweet{}
	_, err := query.GetAll(ctx, &tweets)

	if err != nil {
		return fmt.Errorf("Error fetching tweets: %v", err)
	}

	user, err := getUser(ctx)
	if err != nil {
		return fmt.Errorf("Error fetching user: %v", err)
	}

	// Add to csv file
	headers := []string{
		"id",
		"created",
		"favorites",
		"retweets",
		"ratio",
		"text",
		"url",
		"deleted",
	}

	rows := make([][]string, len(tweets))
	for i, tweet := range tweets {
		rows[i] = []string{
			tweet.IdStr,
			time.Unix(tweet.Created, 0).Format(ARCHIVE_TIME_FORMAT),
			fmt.Sprintf("%v", tweet.Faves),
			fmt.Sprintf("%v", tweet.Rts),
			fmt.Sprintf("%v", tweet.Ratio),
			tweet.Text,
			tweet.Url,
			fmt.Sprintf("%v", tweet.Deleted),
		}
	}

	w.Header().Set("content-disposition", "attachment; filename=\"" + user.ScreenName + "-archive.zip\"")

	zipWriter := zip.NewWriter(w)
	fileWriter, err := zipWriter.Create("tweets.csv")
	if err != nil {
		return fmt.Errorf("Error creating zip: %v", err)
	}

	writer := csv.NewWriter(fileWriter)
	err = writer.Write(headers)
	if err != nil {
		return fmt.Errorf("Error writing headers: %v", err)
	}

	err = writer.WriteAll(rows)
	if err != nil {
		return fmt.Errorf("Error writing rows: %v", err)
	}

	writer.Flush()
	zipWriter.Flush()
	return zipWriter.Close()
}

func archiveImportHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	reader := csv.NewReader(r.Body)

	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("Error parsing csv file: %v", err)
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
			return fmt.Errorf("Error checking tweets from csv file: %v", err)
		}
		log.Infof(ctx, "checked tweets: %v. Storing...", len(tweets))

		err = storeTweets(ctx, tweets)
		if err != nil {
			return fmt.Errorf("Error storing tweets from csv file: %v", err)
		}
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

func tweetHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	params := r.URL.Query()
	id, _ := strconv.Atoi(params.Get("id"))
	tweet := MyTweet{Id: int64(id)}

	if err := datastore.Get(ctx, tweet.GetKey(ctx), &tweet); err != nil {
		//TODO: return statusnotfound
		return fmt.Errorf("Error getting tweet from datastore: %v", err)
	}

	tweetJson, err := json.Marshal(tweet)
	if err != nil {
		return fmt.Errorf("Error marshaling json for tweet: %v", err)
	}

	_, err = w.Write(tweetJson)
	return err
}

func tweetsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	params := r.URL.Query()
	var (
		tweets []MyTweet
		err error
	)

	i, _ := strconv.Atoi(params.Get("page"))
	which := strings.Replace(path.Clean(r.URL.Path), "/tweets/", "", 1)
	_, err = memcache.JSON.Get(ctx, MEMCACHE_TWEETS_KEY + which, &tweets)
	if i > 0 || err != nil {
		switch which {
		case "best":
			tweets, err = getBestTweets(ctx, i)
		case "latest":
			tweets, err = getLatestTweets(ctx, i)
		case "search":
			tweets, err = getSearchTweets(ctx, i, params.Get("search"), params.Get("order"))
		default:
			//TODO: return bad request
			return fmt.Errorf("Error invalid tweet type: %v", which)
		}

		if err != nil {
			return fmt.Errorf("Error getting %v tweets: %v", which, err)
		}

		store := &memcache.Item{
			Key: MEMCACHE_TWEETS_KEY + which,
			Object: tweets,
		}
		memcache.JSON.Set(ctx, store)
	}

	var tweetJson []byte
	tweetJson, err = json.Marshal(tweets)
	if err != nil {
		return fmt.Errorf("Error marshaling json for tweets: %v", err)
	}

	_, err = w.Write(tweetJson)
	return err
}

func userHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user, err := getUser(ctx)

	if err != nil {
		return fmt.Errorf("Error getting user: %v", err)
	}

	var userJson []byte
	userJson, err = json.Marshal(user)
	if err != nil {
		return fmt.Errorf("Error marshaling json for user: %v", err)
	}

	_, err = w.Write(userJson)
	return err
}

func indexOrErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	name := path.Clean(r.URL.Path)
	if r.Method == "GET" {
		reg, _ := regexp.Compile("/tweet/[0-9]+")
		if name == "/" || reg.MatchString(name) {
			indexHandler(ctx, w, r)
		} else if name == "/favicon.ico" {
			w.WriteHeader(http.StatusNotFound)
		} else {
			//TODO: make /error/404 or something...
			http.Redirect(w, r, "/error", http.StatusMovedPermanently)
		}
		return nil
	}
	//TODO: wrong method error
	return fmt.Errorf("Error wrong method: %v", r.Method)
}

func indexHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user, err := getUser(ctx)
	if err != nil {
		return fmt.Errorf("Error fetching user: %v", err)
	}

	page := "html/main.html"
	name := path.Clean(r.URL.Path)
	if name == "/admin" {
		page = "html/admin.html"
	}

	var temp *template.Template
	temp, err = template.ParseFiles(page)
	if err != nil {
		return fmt.Errorf("Error parsing template: %v", err)
	}

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

	if err = temp.Execute(w, mainPage); err != nil {
		return fmt.Errorf("Error executing template: %v", err)
	}
	return nil
}

func unretweetHanlder(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	twitterApi, myToken := LoadCredentials(true)
	twitterApi.HttpClient.Transport = &urlfetch.Transport{Context: ctx}
	tweets := []anaconda.Tweet{}
	vals := url.Values{
		"screen_name": {myToken.ScreenName},
		"count": {"200"},
		"trim_user": {"1"},
		"exclude_replies": {"1"},
		"include_rts": {"1"},
	}
	lastId := int64(0)
	before := time.Now().Unix() - (DAYS_BEFORE_UNRETWEET * SECONDS_IN_DAY)
	log.Infof(ctx, "unretweet tweets before: %v", time.Unix(before, 0))

	for {
		idStr := fmt.Sprintf("%v", lastId - 1)
		if idStr == vals.Get("max_id") {
			log.Warningf(ctx, "same last id: %v", idStr)
			break
		}
		if lastId != 0 {
			vals.Set("max_id", idStr)
		}
		aTweets, err := twitterApi.GetUserTimeline(vals)
		if err != nil {
			return fmt.Errorf("Error getting tweets: %v", err)
		}

		log.Infof(ctx, "Got tweets, %v", len(aTweets))
		if len(aTweets) == 0 {
			break
		} else {
			for _, tweet := range aTweets {
				if lastId == 0 || tweet.Id < lastId {
					lastId = tweet.Id
				}

				time := parseTimestamp(tweet.CreatedAt, SEARCH_TIME_FORMAT)
				if tweet.RetweetedStatus != nil && time.Unix() < before {
					tweets = append(tweets, tweet)
				}
			}
		}
	}

	log.Infof(ctx, "tweets to unretweet: %v", len(tweets))
	for _, tweet := range tweets {
		if _, err := twitterApi.UnRetweet(tweet.Id, true); err != nil {
			log.Warningf(ctx, "Error unretweeting: %v", err)
		}
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

func updateUserHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	if _, err := fetchAndStoreUser(ctx); err != nil {
		return fmt.Errorf("Error fetching and storing user: %v", err)
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

func updateTweetsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	if err := updateDatastoreTweets(ctx); err != nil {
		return fmt.Errorf("Error updating db tweets: %v", err)
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

func fetchTweetsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	if _, err := fetchAndStoreTweets(ctx); err != nil {
		return fmt.Errorf("Error fetching and storing tweets: %v", err)
	}
	w.WriteHeader(http.StatusOK)
	return nil
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

	ext := path.Ext(anacondaUser.ProfileImageUrlHttps)
	media := Media{
		IdStr: "avatar-" + anacondaUser.IdStr,
		Url: anacondaUser.ProfileImageUrlHttps,
		ExpandedUrl: anacondaUser.ProfileImageUrlHttps,
		Type: "text",
		MediaUrl: anacondaUser.ProfileImageUrlHttps,
		UploadFileName: "user/" + anacondaUser.IdStr + "/avatar" + ext,
	}

	if err := fetchAndStoreMediaFile(ctx, media); err != nil {
		return nil, fmt.Errorf("Error fetching and storing user profile pic: %v", err)
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
		Media: media,
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
				t.Media, err = getMedia(ctx, &aTweet)
				if err != nil {
					return nil, err
				}
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

	procTweets, newLastId := processTweets(ctx, aTweets)
	tweets = append(tweets, procTweets...)

	log.Infof(ctx, "Fetched Tweets: %v; (newLastId): %v", len(tweets), newLastId)

	return fetchTweets(ctx, tweets, newLastId, latestId)
}

func processTweets(ctx context.Context, tweets []anaconda.Tweet) ([]MyTweet, int64) {
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
			}
			m, err := getMedia(ctx, &tweet)
			if err != nil {
				log.Warningf(ctx, "Error getting media: %v", err)
			}
			myTweet.Media = m

			out = append(out, myTweet)
		}
	}

	return out, lastId
}

func getMedia(ctx context.Context, tweet *anaconda.Tweet) (media []Media, err error) {
	if len(tweet.Entities.Media) > 0 {
		for i, ent := range tweet.Entities.Media {
			m := Media{
				Type: ent.Type,
				IdStr: ent.Id_str,
				Url: ent.Url,
				ExpandedUrl: ent.Expanded_url,
				MediaUrl: ent.Media_url_https,
			}
			m.UploadFileName = getMediaFilePath(tweet.IdStr, m, i)
			//log.Infof(ctx, "Uploading image path: " + m.UploadFileName + ", %+v", m)
			if err := fetchAndStoreMediaFile(ctx, m); err != nil {
				return nil, fmt.Errorf("Error fetching and storing media file: %v", err)
			}
			media = append(media, m)
		}
	}
	return media, nil
}

func getBucket(ctx context.Context) (*storage.BucketHandle, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("Error getting storage.Client: %v", err)
	}

	bucketName, err := file.DefaultBucketName(ctx)
	if err != nil {
		return nil, fmt.Errorf("Error getting default bucket name: %v", err)
	}

	return client.Bucket(bucketName), nil
}

func fetchAndStoreMediaFile(ctx context.Context, media Media) error {
	bucket, err := getBucket(ctx)
	if err != nil {
		return fmt.Errorf("Error getting bucket: %v", err)
	}

	client := urlfetch.Client(ctx)
	resp, err := client.Get(media.MediaUrl)
	if err != nil {
		return fmt.Errorf("Error fetching media file from twitter: %v", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Error reading response: %v", err)
		}

		log.Infof(ctx, "Storing media file: " + media.UploadFileName)
		if err = storeMediaFile(ctx, bucket, media.UploadFileName, bodyBytes); err != nil {
			log.Errorf(ctx, "Error storing media file: %v", err)
			// send bad request or something
			return fmt.Errorf("Error storing media file to cloud storage: %v", err)
		}
	} else {
		// send back request or something
		log.Warningf(ctx, "Status not ok: %v", resp)
		//return fmt.Errorf("Response status bad: %v", resp)

	}
	return nil
}

func storeMediaFile(ctx context.Context, bucket *storage.BucketHandle, fileName string, content []byte) error {
	wc := bucket.Object(fileName).NewWriter(ctx)
	if _, err := wc.Write(content); err != nil {
		return err
	}

	if err := wc.Close(); err != nil {
		return err
	}

	return nil
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

func getMediaFilePath(tweetID string, m Media, i int) string {
	num := strconv.Itoa(i + 1)
	ext := path.Ext(m.MediaUrl)
	return "status/" + tweetID + "/" + m.Type + "/" + num + ext
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
