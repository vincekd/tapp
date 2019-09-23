const nconf = (module.exports = require('nconf'));
const path = require('path');

nconf.argv()
     .env(['NODE_ENV', 'PORT'])
     .file({file: path.join(__dirname, 'config.json')})
     .defaults({
       PORT: 8080,
     });
