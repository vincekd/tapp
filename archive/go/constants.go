package tapp

const (
	TWITTER_URL = "https://twitter.com/"
	MEMCACHE_TWEETS_KEY = "TWEETS."
	MEMCACHE_USER_KEY = "USER."
	TWEETS_TO_FETCH = 30
	MIN_RATIO = float32(0.10)
	MAX_PUT_SIZE int = 500
	MAX_API_LOOKUP_SIZE int = 100
	MIN_SEARCH_LENGTH int = 2
	SEARCH_TIME_FORMAT = "Mon Jan 2 15:04:05 -0700 2006"
	ARCHIVE_TIME_FORMAT = "2006-01-02 15:04:05 -0700"
	XML_ATOM_TIME_FORMAT = "2006-01-02T15:04:05Z"
	FEED_HEADER_FORMAT = "15:04:05 2006-01-02"
	SUMMARY_LENGTH = 30
	SECONDS_IN_DAY = int64(86400)
	DAYS_BEFORE_UNRETWEET = 6
)
