'use strict';

const path = require('path');
const express = require('express');
const config = require('./config');
const StatusCodes = require("http-status-codes");
const { Feed } = require("feed");

const userServ = require("./services/user.service.js");
const tweetServ = require("./services/tweet.service.js");
const mediaServ = require("./services/media.service.js");
const twitterServ = require("./services/twitter.service.js");

const {
  TWITTER_URL,
  SUMMARY_LENGTH,
} = require("./constants.js");

const app = express();

app.disable('etag');
app.set('views', path.join(__dirname, 'views'));
app.set('view engine', 'pug');
app.set('trust proxy', true);

// Redirect root to /books
// app.get('/', (req, res) => {
//   //res.redirect('/books');
// });

//app.use("/dist", express.static('dist'));
app.use("/js/", express.static("dist/js"));
app.use("/css/", express.static("dist/css"));
app.use("/media/", express.static('media'));
app.use("/templates/", express.static("templates"));
app.get("/service-worker.js", (req, res) => {
  res.sendFile(path.join(__dirname, 'dist/js/service-worker.js'));
});
app.get("/favicon.ico", (req, res) => {
  res.status(StatusCodes.NOT_FOUND).send();
});

app.get(["/", "/index.html", "/index", "/latest", "/best", "/search", "/error", "/tweet/:tweetID"], (req, res) => {
  userServ.get(config.get("screenName")).then(user => {
    const gaKey = config.get("gaTrackingId");
    res.render("index",  {
      user,
      gaKey,
      isDev: isDev(),
    });
  });
});

// AJAX Calls
app.get("/user", handleAjax((req, res) => userServ.get(config.get("screenName"))));
app.get("/tweet", handleAjax((req, res) => tweetServ.get(req.query.id)));
app.get("/tweets/:which", handleAjax((req, res) => {
  switch (req.params.which) {
    case "best":
      return tweetServ.getBest(req.query.page);
    case "latest":
      return tweetServ.getLatest(req.query.page);
    case "search":
      return tweetServ.getSearch(req.query.page, req.query.search, req.query.order);
    default:
      throw { code: StatusCodes.BAD_REQUEST, };
  }
}));

app.get("/media", (req, res) => {
  mediaServ.get(req.query.file).then(file => {
    file.createReadStream().on("error", err => error(res, err)).pipe(res);
  }).catch(e => error(res, e));
});

// CRON calls
app.get("/fetch", validateCron(async (req, res) => {
  try {
    const lastTweet = await tweetServ.getLast();
    const minID = lastTweet? lastTweet.IdStr : null;
    const tweets = await twitterServ.getNewTweets(config.get("screenName"), minID);
    await Promise.all(tweets.map(t => {
      if (t.Media.length) {
        return mediaServ.fetchAndStoreTweetMedia(t);
      }
      return Promise.resolve(true);
    }));

    await tweetServ.save(tweets);
    res.status(StatusCodes.OK).send();
  } catch (e) {
    console.error("error in fetch", e);
    error(res, e);
  }
}));
app.get("/update/tweets", validateCron(async (req, res) => {
  try {
    await tweetServ.checkAndUpdateTweets();
    res.status(StatusCodes.OK).send();
  } catch (e) {
    console.error("error in update/tweets", e);
    error(res, e);
  }
}));
app.get("/update/user", validateCron(async (req, res) => {
  try {
    const user = await twitterServ.getUser(config.get("screenName"));

    await mediaServ.fetchAndStore(user.Media);
    await userServ.save(user);

    res.status(StatusCodes.OK).send();
  } catch (e) {
    console.error("error in update/user", e);
    error(res, e);
  }
}));

// xml
app.get("/feed/latest.xml", handleAjax(async (req, res) => {
  const name = config.get("screenName");
  const tweets = await tweetServ.getLatest(0);
  const user = await userServ.get(name);

  const author = {
    name: `@${name}`,
    link: `${user.Url}`,
  };
  const feed = new Feed({
    title: "@" + name + " Latest Tweets",
    description: "Latest tweets by @" + name,
    id: config.get("siteUrl"),
    link: config.get("siteUrl"),
    language: "en",
    //image: "http://example.com/image.png",
    //favicon: "http://example.com/favicon.ico",
    copyright: `All rights reserved ${(new Date()).getFullYear()}, @${name}`,
    author,
  });

  tweets.forEach(tweet => {
    feed.addItem({
      title: tweet.Text.substring(0, SUMMARY_LENGTH) + "...",
      id: tweet.IdStr,
      link: tweet.Url,
      description: "A Tweet",
      content: tweet.Text,
      author: [author],
      date: new Date(tweet.Created * 1000),
      //image: "asdf"
    });
  });

  res.setHeader("Content-Type", "application/atom+xml; charset=utf-8");
  res.status(StatusCodes.OK).send(feed.atom1());
}));

// app.get("/dedupe", async (req, res) => {
//   const tweets = await tweetServ.getAll();
//   const map = {};
//   tweets.forEach(t => {
//     map[t.IdStr] = map[t.IdStr] || [];
//     map[t.IdStr].push(t);
//   });
//   const dupes = [];
//   Object.keys(map).forEach(key => {
//     if (map[key].length > 1) {
//       dupes.push(map[key]);
//     }
//   });

//   const toRemove = [];
//   dupes.forEach(dupe => {
//     dupe.forEach(item => {
//       const key = item[Object.getOwnPropertySymbols(item)[0]];
//       if (key && key.name && !key.id) {
//         toRemove.push({item, key});
//       }
//     });
//   });

//   const ds = require("./services/datastore.service.js");
//   //await tweetServ.remove(toRemove);
//   ds.ds.delete(toRemove.map(a => a.key), (err) => {
//     if (err) {
//       console.error(err);
//     } else {
//       res.status(StatusCodes.OK).send();
//     }
//   });
// });

// app.get("/unretweet", validateCron((req, res) => {

// }));

function validateCron(handler) {
  return (req, res) => {
    if (req.get("X-Appengine-Cron") != "true" && !isDev()) {
      return error(res, { code: StatusCodes.UNAUTHORIZED });
    }
    return handler(res, res);
  };
}

function handleAjax(handler) {
  return async (req, res) => {
    try {
      const out = await handler(req, res);
      res.json(out);
    } catch (e) {
      console.error(e);
      error(res, e);
    }
  };
}

function error(res, err) {
  console.error(err);
  try {
    res.status(err.code || StatusCodes.INTERNAL_SERVER_ERROR).send();
  } catch (e) {
    res.status(StatusCodes.INTERNAL_SERVER_ERROR).send();
  }
}

function isDev() {
  return config.get("NODE_ENV") === "development";
}
// Basic 404 handler
// app.use((req, res) => {
//   //res.status(404).send('Not Found');
// });

// Basic error handler
// app.use((err, req, res) => {
//   console.error(err);
//   res.status(StatusCodes.INTERNAL_SERVER_ERROR).send(err.response || 'Something broke!');
// });

if (module === require.main) {
  // Start the server
  const server = app.listen(config.get('PORT'), () => {
    const port = server.address().port;
    console.log(`App listening on port ${port}`);
  });
}

module.exports = app;