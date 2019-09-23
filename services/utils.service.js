
class Utils {
  normalizeMedia(entity) {
    const copy = Object.assign({}, entity);
    const media = {};
    Object.keys(entity).forEach(key => {
      if (key.startsWith("Media.")) {
        media[key.replace(/^Media\./, "")] = entity[key];
        delete copy[key];
      }
    });
    if (Object.keys(media).length > 0) {
      copy.Media = media;
    }
    return copy;
  }
}

module.exports = new Utils();
