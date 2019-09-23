const { Datastore } = require("@google-cloud/datastore");
const utils = require("./utils.service.js");
const ds = require("./datastore.service.js");
const twitterServ = require("./twitter.service.js");

const {
  TWEETS_TO_FETCH,
  TWEETS_KEY,
  MIN_SEARCH_LENGTH,
  MAX_API_LOOKUP_SIZE,
  HAS_MORE,
} = require("../constants.js");

class TweetService {
  tweetID(id) {
    return Datastore.int(id);
  }

  normalize(tweet) {
    const t = utils.normalizeMedia(tweet);
    if (!t.Media) {
      t.Media = [];
    } else if (!Array.isArray(t.Media)) {
      t.Media = [t.Media];
    }
    return t;
  }

  save(tweets) {
    const items = tweets.map(tweet => ({
      id: this.tweetID(tweet.IdStr),
      data: tweet,
    }));
    return ds.putAll(TWEETS_KEY, items);
  }

  async get(id) {
    const tweet = await ds.get(TWEETS_KEY, this.tweetID(id));
    return this.normalize(tweet);
  }

  async run(query) {
    const tweets = await ds.run(query);
    return tweets.map(this.normalize);
  }

  async getLast() {
    const query = ds.Query(TWEETS_KEY).
          order("Id", { descending: true }).
          limit(1);
    const [tweet] = await ds.run(query);
    return tweet;
  }

  async checkAndUpdateTweets() {
    // get all tweets from db
    let tweets = await this.getAll();
    // check if tweets have changed
    tweets = await this.checkTweets(tweets);
    // // re-store updated ones
    return this.save(tweets);
  }

  async checkTweets(tweets) {
    const toCheck = tweets.slice(0, MAX_API_LOOKUP_SIZE);
    const fetched = await twitterServ.getTweets(toCheck.map(t => t.IdStr));
    // checking...
    const changed = [];
    toCheck.forEach(tweet => {
      const t = fetched.find(t => t.IdStr === tweet.IdStr);
      if (!t) {
        tweet.Deleted = true;
        changed.push(tweet);
      } else if (t.Faves !== tweet.Faves || t.Rts !== tweet.Rts) {
        tweet.Faves = t.Faves;
        tweet.Rts = t.Rts;
        changed.push(tweet);
      }
    });

    const rest = tweets.slice(MAX_API_LOOKUP_SIZE);
    if (rest.length > 0) {
      return changed.concat(await this.checkTweets(rest));
    }
    return changed;
  }

  async getAll(includeDeleted = false) {
    let query = ds.Query(TWEETS_KEY).order("Updated");
    if (!includeDeleted) {
      query = query.filter("Deleted", "=", false);
    }

    const out = await ds.run(query);
    if (out[HAS_MORE]) {
      // TODO: has more
      console.warn("HAS MORE ENTITIES", out.length);
    }
    return out.map(this.normalize);
  }

  getBest(page) {
    const query = ds.Query(TWEETS_KEY).
		  filter("Deleted", "=", false).
		  order("Faves", { descending: true }).
		  order("Rts", { descending: true }).
		  order("Ratio", { descending: true }).
		  limit(TWEETS_TO_FETCH).
		  offset(page * TWEETS_TO_FETCH);
    return this.run(query);
  }

  getLatest(page) {
    const query = ds.Query(TWEETS_KEY).
		  filter("Deleted", "=", false).
		  order("Id", { descending: true }).
		  limit(TWEETS_TO_FETCH).
		  offset(page * TWEETS_TO_FETCH);
    return this.run(query);
  }

  getSearch(page, search = "", order = "-Faves") {
    search = this.removePunctuation(search.trim(), false);
    if (search.length > MIN_SEARCH_LENGTH) {
      const terms = this.getTerms(search);
      if (terms.length > 0) {
        const descending = order.startsWith("-");
        const num = page * TWEETS_TO_FETCH;
        const q = ds.Query(TWEETS_KEY).
              filter("Deleted", "=", false).
              order(descending ? order.substring(1) : order, { descending }).
              offset(num);
        return this.run(q).then(tweets => {
          return this.searchTweets(tweets, terms).slice(num, num + TWEETS_TO_FETCH);
        });
      }
    }

    return new Promise.resolve([]);
  }

  searchTweets(tweets, terms) {
    return tweets.filter(tweet => {
      const text = this.removePunctuation(tweet.Text, true).toUpperCase();
      return terms.some(and => {
        return and.every(term => {
          if (term.quoted && term.regexp) {
            return term.regexp.test(text);
          }
          return text.indexOf(term.upper) > -1;
        });
      });
    });
  }

  getTerms(search) {
    return search.split(" OR ").reduce((terms, or) => {
      const split = or.split(/[^\S]+/);
      const and = [];
      split.forEach(term => {
        if (term.length > MIN_SEARCH_LENGTH) {
          const o = {
            original: term,
            quoted: term.startsWith("\"") && term.endsWith("\""),
			text: term,
			regexp: null,
          };
          if (o.quoted) {
            o.text = term.substring(1, term.length - 1);
            o.regexp = new RegExp("(^| )" + o.upper + "( |$)");
		  } else {
            o.text = this.removePunctuation(term, true);
          }
          o.upper = o.text.toUpperCase();
          and.push(o);
        }
      });

      if (and.length > 0) {
        terms.push(and);
      }
      return terms;
    }, []);
  }

  removePunctuation(str = "", quotes = false) {
	if (quotes) {
      return str.replace(/[^a-zA-Z0-9 ]/g, "");
	}
    return str.replace(/[^a-zA-Z0-9 \"]/g, "");
  }
}

module.exports = new TweetService();
