const ds = require("./datastore.service.js");
const utils = require("./utils.service.js");
const {
  USERS_KEY,
} = require("../constants.js");

class UserService {
  async get(screenName) {
    // TODO: IF null, fetch and save
    return utils.normalizeMedia(await ds.get(USERS_KEY, screenName));
  }

  save(user) {
    return ds.put(USERS_KEY, user.ScreenName, user);
  }
}

module.exports = new UserService();
