
const VERSION = "v0.0.5";
const RESOURCE_CACHE = "tapp.resources." + VERSION;
const USER_CACHE = "tapp.user." + VERSION;
const TWEET_CACHE = "tapp.tweets." + VERSION;
const CACHE_WHITELIST = [RESOURCE_CACHE, USER_CACHE, TWEET_CACHE];
const USER_URL = "/user";
const TWEET_REGEXP = /\/tweets\/(?:best|latest)(?:\\?.*){0,1}$/i;

const CACHED_RESOURCES = [
    // CSS
    "/media/twitter-fontello/css/tweet-icons.css",
    "/css/theme.css",
    "/css/main.css",
    // JS
    "/dist/js/app.min.js",
    // HTML
    "/templates/menu.html",
    "/templates/tweets.html",
    "/templates/search.html",
    "/templates/tweet.html",
    "/templates/error.html",
    // MISC
    "/media/twitter-fontello/font/tweet-icons.woff2?27142445",
    // "/media/twitter-fontello/font/tweet-icons.eot?27142445",
    // "/media/twitter-fontello/font/tweet-icons.eot?27142445#iefix",
    // "/media/twitter-fontello/font/tweet-icons.woff?27142445",
    // "/media/twitter-fontello/font/tweet-icons.ttf?27142445",
    // "/media/twitter-fontello/font/tweet-icons.svg?27142445#tweet-icons",
    ".jpg",
    ".png",
];

// self.addEventListener("message", function() {
//     console.log("message", arguments);
// });
self.addEventListener("error", function(err) {
    console.error("Error in service worker", err);
});

self.addEventListener("fetch", function(event: any) {
    const isResource = CACHED_RESOURCES.some(r => event.request.url.endsWith(r));
    const isUser = event.request.url.endsWith(USER_URL);
    const isTweets = TWEET_REGEXP.test(event.request.url);
    if (isResource || isUser || isTweets) {
        event.respondWith(caches.match(event.request).then(resp => {
            if (isResource) {
                return resp || fetch(event.request).then(resp => {
                    return caches.open(RESOURCE_CACHE).then(cache => {
                        cache.put(event.request, resp.clone());
                        return resp;
                    });
                });
            } else {
                // eventually fresh
                let req = fetch(event.request).then(resp => {
                    return caches.open(isUser ? USER_CACHE : TWEET_CACHE).then(cache => {
                        cache.put(event.request, resp.clone());
                        return resp;
                    });
                });

                return resp || req;
            }
        }));
    }
});

self.addEventListener("activate", function(event: any) {
    console.info("activating service worker");
    event.waitUntil(
        caches.keys().then(
            keyList => Promise.all(keyList.map(key => {
                if (CACHE_WHITELIST.indexOf(key) < 0) {
                    console.log("deleting cache", key);
                    return caches.delete(key);
                }
                return true;
            }))
        )
    );
});

self.addEventListener("install", function(event: any) {
    console.log("installing service worker");
    event.waitUntil(
        Promise.all([
            caches.open(RESOURCE_CACHE).then(cache => cache.addAll(CACHED_RESOURCES.filter(r => r.startsWith("/")))),
            caches.open(USER_CACHE).then(cache => cache.addAll([USER_URL])),
        ])
    );
});
