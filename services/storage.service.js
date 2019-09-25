const { Storage } = require("@google-cloud/storage");
const StatusCodes = require("http-status-codes");
const config = require("../config.js");

class StorageService {
  constructor() {
    this.storage = new Storage();
    let bucketName = config.get("bucketName");
    // if (config.get("NODE_ENV") === "development") {
    //   bucketName = "staging."+ bucketName;
    // }
    this.bucket = this.storage.bucket(bucketName);
  }

  get(path) {
    return new Promise((resolve, reject) => {
      const file = this.bucket.file(path);
      file.exists((err, exists) => {
        if (err) {
          reject(err);
        } else if (!exists) {
          reject({
            code: StatusCodes.NOT_FOUND,
            message: 'Not found',
          });
        } else {
          resolve(file);
        }
      })
    });
  }

  async put(path, data) {
    return new Promise((resolve, reject) => {
      const file = this.bucket.file(path);
      file.save(data, (err) => {
        if (err) {
          reject(err);
        } else {
          resolve(true);
        }
      });
    });
  }
}

module.exports = new StorageService();
