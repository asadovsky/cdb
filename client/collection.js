// Collection class.

var Conn = require('./conn');
var util = require('./util');

module.exports = Collection;

function Collection(addr, name) {
  this.addr_ = addr;
  this.name_ = name;
  // Map of key to CValue, populated from watch stream.
  this.data_ = {};
}

// Opens this collection, initiating the watch stream.
// TODO: Eliminate this method once we've implemented fine-grained watch.
Collection.prototype.open = function(cb) {
  var that = this;

  // Initialized by processSubscribeResponseS2C_.
  this.agentId_ = null;
  this.clientId_ = null;

  // Initialize connection.
  var conn = new Conn(this.addr_);

  conn.on('open', function() {
    conn.send({
      Type: 'SubscribeC2S'
    });
  });

  conn.on('recv', function(msg) {
    switch (msg.Type) {
    case 'SubscribeResponseS2C':
      return that.processSubscribeResponseS2C_(msg);
    case 'ValueS2C':
      return that.processValueS2C_(msg);
    case 'PatchS2C':
      return that.processPatchS2C_(msg);
    default:
      throw new Error('unknown message type: ' + msg.Type);
    }
  });
};

Collection.prototype.processSubscribeResponseS2C_ = function(msg) {
  this.agentId_ = msg.AgentId;
  this.clientId_ = msg.ClientId;
};

Collection.prototype.processValueS2C_ = function(msg) {
  throw new Error('FIXME');
};

Collection.prototype.processPatchS2C_ = function(msg) {
  throw new Error('FIXME');
};

// Creates this collection. If opts.failIfExists is set, fails if the collection
// already exists.
Collection.prototype.create = function(opts, cb) {
  util.setImmediate(function() {
    cb(new Error('not implemented'));
  });
};

// Destroys this collection. If opts.failIfMissing is set, fails if the
// collection does not exist.
Collection.prototype.destroy = function(opts, cb) {
  util.setImmediate(function() {
    cb(new Error('not implemented'));
  });
};

// Gets the CValue for the given key. If opts.dtype is specified, checks that
// the value has the given dtype.
Collection.prototype.get = function(key, opts, cb) {
  util.setImmediate(function() {
    cb(new Error('FIXME'));
  });
};

// Gets the CValue for the given key. If the value already exists, checks that
// it has the given dtype; otherwise, creates it with the given dtype.
Collection.prototype.getOrCreate = function(key, dtype, opts, cb) {
  util.setImmediate(function() {
    cb(new Error('FIXME'));
  });
};

// Puts the given value for the given key. Value must be a native JS type, and
// will be converted to a CRegister.
Collection.prototype.put = function(key, value, opts, cb) {
  util.setImmediate(function() {
    cb(new Error('FIXME'));
  });
};

// Deletes the specified record. If opts.failIfMissing is set, fails if there is
// no record with the given key.
Collection.prototype.del = function(key, opts, cb) {
  util.setImmediate(function() {
    cb(new Error('not implemented'));
  });
};
