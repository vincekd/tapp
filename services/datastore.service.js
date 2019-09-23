const { Datastore } = require('@google-cloud/datastore');
const StatusCodes = require("http-status-codes");

const {
  MAX_PUT_SIZE,
  HAS_MORE,
} = require("../constants.js");

class TAppDataStore {
  constructor() {
    this.ds = new Datastore();
  }

  putAll(kind, items) {
    return new Promise((resolve, reject) => {
      const slice = items.slice(0, MAX_PUT_SIZE);
      const entities = slice.map(item => {
        return {
          key: this.ds.key([kind, item.id]),
          data: item.data,
        };
      });

      this.ds.save(entities, (err) => {
        if (err) {
          reject(err);
        } else {
          if (items.slice(MAX_PUT_SIZE).length > 0) {
            resolve(this.putAll(kind, items.slice(MAX_PUT_SIZE)));
          } else {
            resolve(true);
          }
        }
      });
    });
  }

  put(kind, id, data) {
    return new Promise((resolve, reject) => {
      const key = this.ds.key([kind, id]);
      this.ds.save({ key, data }, (err) => {
        if (err) {
          reject(err);
        } else {
          resolve(true);
        }
      });
    });
  }

  get(kind, id) {
    return new Promise((resolve, reject) => {
      const key = this.ds.key([kind, id]);
      this.ds.get(key, (err, entity) => {
        if (err) {
          reject(err);
        } else if (!entity) {
          reject({
            code: StatusCodes.NOT_FOUND,
            message: 'Not found',
          });
        } else {
          resolve(this.fromDatastore(entity));
        }
      });
    });
  }

  Query(kind) {
    return this.ds.createQuery(kind);
  }

  run(query) {
    return new Promise((resolve, reject) => {
      this.ds.runQuery(query, (err, entities, nextQuery) => {
        if (err) {
          reject(err);
        } else {
          // HACK: assign hasMore to array
          const out = entities.map(e => this.fromDatastore(e));
          out[HAS_MORE] = nextQuery.moreResults !== Datastore.NO_MORE_RESULTS ? nextQuery.endCursor : false;
          resolve(out);
        }
      });
    });
  }

  fromDatastore(obj) {
    obj.id = obj[Datastore.KEY].id;
    return obj;
  }
}

module.exports = new TAppDataStore();
