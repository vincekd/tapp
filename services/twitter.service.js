const path = require("path");
const BigNumber = require("big-number");
const Twit = require("twit");

const config = require("../config.js");
const {
  TWITTER_URL,
} = require("../constants.js");

class TwitterService {
  constructor() {
    this.twit = new Twit({
      app_only_auth: true,
      consumer_key: config.get("twitter:consumerKey"),
      consumer_secret: config.get("twitter:consumerKeySecret"),
      strictSSL: true,
    });
  }

  // INFO: `from:username` is broken in twitter's api...
  search(name, search, order) {
    return new Promise((resolve, reject) => {
      const params = {
        q: `from:${name} -filter:replies ${search}`,
        result_type: order === "Id" ? "recent" : "popular",
        count: 100,
        include_entities: 1,
        tweet_mode: "extended",
        // TODO: implement
        //max_id: maxid
      };
      console.log("params", params);

      this.twit.get("search/tweets", params, (err, data, resp) => {
        if (err) {
          reject(err);
        } else {
          console.log("data", data);
          resolve(data.statuses.map(t => this.normalizeTwitterTweet(t)));
        }
      });
    });
  }

  getTweets(ids) {
    return new Promise((resolve, reject) => {
      const params = {
        id: ids.join(","),
        trim_user: 1,
        include_entities: 1,
        tweet_mode: "extended",
      };

      this.twit.get("statuses/lookup", params, async (err, data, resp) => {
        if (err) {
          reject(err);
        } else {
          resolve(data.map(t => this.normalizeTwitterTweet(t)));
        }
      });
    });
  }

  getNewTweets(screenName, minIDStr, maxIDStr) {
    return new Promise((resolve, reject) => {
      const params = {
        screen_name: screenName,
        count: 200,
        trim_user: 1,
        exclude_replies: 1,
        include_rts: 0,
        tweet_mode: "extended",
      };

      if (minIDStr) {
        params.since_id = minIDStr;
      }

      if (maxIDStr) {
        params.max_id = BigNumber(maxIDStr).minus(1).toString();
      }

      this.twit.get("statuses/user_timeline", params, async (err, data, resp) => {
        if (err) {
          reject(err);
        } else {
          if (data.length > 0) {
            const [tweets, newMax] = this.processNewTweets(data);
            const more = await this.getNewTweets(screenName, minIDStr, newMax);
            resolve(tweets.concat(more));
          } else {
            resolve(data);
          }
        }
      });
    });
  }

  getUser(screenName) {
    return new Promise((resolve, reject) => {
      const params = {
        screen_name: screenName,
        include_entities: 1,
      };
      this.twit.get("users/lookup", params, (err, data, resp) => {
        if (err) {
          reject(err);
        } else if (data.length === 0) {
          reject({ code: StatusCodes.NOT_FOUND });
        } else {
          resolve(this.normalizeTwitterUser(data[0]));
        }
      });
    });
  }

  processNewTweets(tweets) {
    let last = BigNumber(0);
    const out = [];
    tweets.forEach(tweet => {
      if (last.equals(0) || BigNumber(tweet.id_str).lt(last)) {
        last = BigNumber(tweet.id_str);
      }

      if (this.testTweet(tweet)) {
        out.push(this.normalizeTwitterTweet(tweet));
      }
    });
    return [out, last.toString()];
  }

  normalizeTwitterTweet(t) {
    const tweet = {
      Ratio: this.getRatio(t.favorite_count, t.retweet_count),
	  IdStr: t.id_str,
	  Faves: t.favorite_count,
	  Rts: t.retweet_count,
	  Id: t.id,
	  Created: this.parseTimestamp(t.created_at),
	  Updated: Math.round(Date.now() / 1000),
	  Text: t.full_text,
	  Url: TWITTER_URL + config.get("screenName") + "/status/" + t.id_str,
	  Deleted: false,
      Media: [],
    };

    if (t.entities.media && t.entities.media.length) {
      tweet.Media = t.entities.media.map((ent, i) => {
        return {
          Type: ent.type,
		  IdStr: ent.id_str,
		  Url: ent.url,
		  ExpandedUrl: ent.expanded_url,
		  MediaUrl: ent.media_url_https,
          UploadFileName: "status/" + t.id_str + "/" + ent.type + "/" +
                (i + 1) + path.extname(ent.media_url_https),
        };
      });
    }

    return tweet;
  }

  normalizeTwitterUser(u) {
    const imgUrl = u.profile_image_url_https;
    const rev = imgUrl.match(/([0-9]+)\/.+\.[a-z]+$/)[1];
    const entities = (u.entities || {});
    return {
      Id: u.id,
      IdStr: u.id_str,
      ScreenName: u.screen_name,
      Url: TWITTER_URL + u.screen_name,
	  ProfileImageUrlHttps: imgUrl,
	  Name: u.name,
	  Description: this.replaceLinks(u.description, entities.description),
	  Followers: u.followers_count,
	  Following: u.friends_count,
	  TweetCount: u.statuses_count,
	  Location: u.location,
	  Verified: u.verified,
      Link: this.replaceLinks(u.url, entities.url),
      Updated: Math.round(Date.now() / 1000),
      CreatedAt: u.created_at,
      Media: {
        IdStr: "avatar-" + u.id_str,
		Url: imgUrl,
		ExpandedUrl: imgUrl,
		Type: "text",
		MediaUrl: imgUrl,
		UploadFileName: `user/${u.id_str}/avatar-${rev}${path.extname(imgUrl)}`,
      }
    };
  }

  replaceLinks(str, o) {
    if (!str) {
      return "";
    }
    if (!o || !o.urls) {
      return str;
    }
    o.urls.forEach(u => {
      const re = new RegExp("(^|\\b)" + u.url.replace(".", "\\.") + "(\\b|$)", "ig");
      str = str.replace(re, u.expanded_url);
    });
    return str;
  }

  getRatio(favs, rts) {
	if (favs <= 0) {
	  return 0;
	}
	return (rts / favs);
  }

  parseTimestamp(created) {
    return (new Date(created)).getTime() / 1000;
  }

  testTweet(tweet) {
    // return tweet.id_str &&
	// 	  tweet.entities.user_mentions.length === 0 &&
	// 	  tweet.entities.urls.length === 0;
    return true;
  }
}

module.exports = new TwitterService();
