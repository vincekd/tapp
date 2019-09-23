const https = require("https");
const StatusCodes = require("http-status-codes");

const storage = require("./storage.service.js");

class MediaService {
  get(path) {
    return storage.get(path);
  }

  fetchAndStoreTweetMedia(tweet) {
    return Promise.all(tweet.Media.map(m => this.fetchAndStore(m)));
  }

  async fetchAndStore(media) {
    const buffer = await this.fetch(media);
    return storage.put(media.UploadFileName, buffer);
  }

  fetch(media) {
    return new Promise((resolve, reject) => {
      // FIXME: don't wanna replace _normal when not profile avatar
      https.get(media.MediaUrl.replace("_normal.", "_400x400."), (res) => {
        if (res.statusCode !== StatusCodes.OK) {
          reject({ code: res.statusCode });
        } else {
          const data = [];
          res.on('error', e => reject(e))
             .on('data', chunk => data.push(chunk))
             .on('end', () => {
               resolve(Buffer.concat(data));
             });
        }
      });
    });
  }
}

module.exports = new MediaService();
