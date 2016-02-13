// Store class.

var Collection = require('./collection');

module.exports = Store;

function Store(addr) {
  this.addr_ = addr;
}

// Returns a handle for the named collection.
Store.prototype.getCollection = function(name, cb) {
  return new Collection(this.addr_, name);
};
