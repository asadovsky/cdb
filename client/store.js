// Store class.

var _ = require('lodash');

var Conn = require('./conn');
var cvalue = require('./dtypes/cvalue');
var util = require('./dtypes/util');

module.exports = Store;

function Store(addr) {
  this.addr_ = addr;
  // Map of key to CValue, populated from watch stream.
  this.m_ = {};
}

// Opens this store, initiating the watch stream.
// TODO: Eliminate this method once we've implemented fine-grained watch.
Store.prototype.open = function(cb) {
  var that = this;

  // Initialize connection.
  this.conn_ = new Conn(this.addr_);

  this.conn_.on('open', function() {
    that.conn_.send({
      Type: 'SubscribeC2S'
    });
  });

  this.conn_.on('recv', function(msg) {
    switch (msg.Type) {
    case 'ValueS2C':
      return that.processValueS2C_(msg);
    case 'ValuesDoneS2C':
      return cb();
    case 'PatchS2C':
      return that.processPatchS2C_(msg);
    default:
      throw new Error('unknown message type: ' + msg.Type);
    }
  });
};

Store.prototype.putAndWatch_ = function(key, dtype, value) {
  var that = this;
  this.m_[key] = value;
  value.on('patch', function(patch) {
    that.conn_.send({
      Type: 'PatchC2S',
      Key: key,
      DType: dtype,
      Patch: patch
    });
  });
};

Store.prototype.processValueS2C_ = function(msg) {
  this.putAndWatch_(msg.Key, msg.DType, util.decodeValue(msg.DType, msg.Value));
};

Store.prototype.processPatchS2C_ = function(msg) {
  if (msg.DType === cvalue.dtypeDelete) {
    throw new Error('not implemented');
  }
  var hasKey = _.has(this.m_, msg.Key);
  var value = hasKey ? this.m_[msg.Key] : util.newZeroValue(msg.DType);
  value.applyPatch(msg.IsLocal, msg.Patch);
  if (!hasKey) {
    this.putAndWatch_(value);
  }
};

function checkDType(got, want) {
  if (got !== want) {
    throw new Error('wrong dtype: got ' + got + ', want ' + want);
  }
}

// Gets the CValue for the given key. If opts.dtype is specified, checks that
// the value has the given dtype.
Store.prototype.get = function(key, opts) {
  opts = opts || {};
  var value = this.m_[key];
  if (value === undefined) {
    throw new Error('not found: ' + key);
  }
  if (opts.dtype) {
    checkDType(value.dtype(), opts.dtype);
  }
  return value;
};

// Gets the CValue for the given key. If the value already exists, checks that
// it has the given dtype; otherwise, creates it with the given dtype.
Store.prototype.getOrCreate = function(key, dtype, opts) {
  opts = opts || {};
  var hasKey = _.has(this.m_, key), value;
  if (hasKey) {
    value = this.m_[key];
    checkDType(value.dtype(), dtype);
  } else {
    value = util.newZeroValue(dtype);
    this.putAndWatch_(key, dtype, value);
  }
  return value;
};

// Puts the given value for the given key. Value must be a native JS type, and
// will be converted to a CRegister.
Store.prototype.put = function(key, value, opts) {
  opts = opts || {};
  this.getOrCreate(key, 'cregister', {}).set(value);
};

// Deletes the specified record. If opts.failIfMissing is set, fails if there is
// no record with the given key.
Store.prototype.del = function(key, opts) {
  throw new Error('not implemented');
};
