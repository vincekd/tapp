const VERSION = "v2.0.2";
const RESOURCE_CACHE = "tapp.resources." + VERSION;
const USER_CACHE = "tapp.user." + VERSION;
const CACHE_WHITELIST = [RESOURCE_CACHE, USER_CACHE];
const USER_URL = "/user";
const MEDIA_URL = "/media";
const CACHED_RESOURCES = [
  // CSS
  "/assets/twitter-fontello/css/tweet-icons.css",
  "/css/theme.css",
  "/css/main.css",
  // JS
  "/js/app.min.js",
  // HTML
  "/templates/menu.html",
  "/templates/tweets.html",
  "/templates/search.html",
  "/templates/tweet.html",
  "/templates/error.html",
  // MISC
  "/assets/twitter-fontello/font/tweet-icons.woff2",
];
const suffixes = [
  ".jpg",
  ".jpeg",
  ".png",
  ".eot",
  ".ttf",
  ".svg",
  ".woff",
  ".gif",
];

self.addEventListener("error", err => {
  console.error("Error in service worker", err);
});

self.addEventListener("fetch", (event: any) => {
  const url = new URL(event.request.url);
  const isMedia = url.pathname === MEDIA_URL;
  const isResource = (CACHED_RESOURCES.indexOf(url.pathname) > -1 ||
                      (!isMedia && suffixes.some(s => url.pathname.endsWith(s))));
  const isUser = url.pathname === USER_URL;
  if (isResource || isUser || isMedia) {
    event.respondWith(caches.match(event.request, {ignoreSearch: isResource}).then(resp => {

      if (isResource || isMedia) {
        return resp || fetch(event.request).then(resp => {
          return caches.open(RESOURCE_CACHE).then(cache => {
            cache.put(event.request, resp.clone());
            return resp;
          });
        });
      } else {

        // eventually fresh
        const req = fetch(event.request).then(resp => {
          return caches.open(USER_CACHE).then(cache => {
            cache.put(event.request, resp.clone());
            return resp;
          });
        });

        return resp || req;
      }
    }));
  }
});

self.addEventListener("activate", (event: any) => {
  console.info("activating service worker");
  event.waitUntil(
    caches.keys().then(
      keyList => Promise.all(keyList.map(key => {
        if (CACHE_WHITELIST.indexOf(key) < 0) {
          console.info("deleting cache", key);
          return caches.delete(key);
        }
        return true;
      }))
    )
  );
});

self.addEventListener("install", (event: any) => {
  console.info("installing service worker");
  event.waitUntil(
    Promise.all([
      caches.open(RESOURCE_CACHE).then(cache => cache.addAll(CACHED_RESOURCES)),
      caches.open(USER_CACHE).then(cache => cache.addAll([USER_URL])),
    ])
  );
});
