// CRegister class.
// Mostly mirrors server/dtypes/cregister/cregister.go.

var inherits = require('inherits');

var cvalue = require('./cvalue');

////////////////////////////////////////////////////////////
// Events

inherits(Set, cvalue.Event);
function Set(isLocal, value) {
  cvalue.Event.call(this, isLocal);
  this.value = value;
}

////////////////////////////////////////////////////////////
// CRegister

inherits(CRegister, cvalue.CValue);
function CRegister(agentId, vec, time, val) {
  cvalue.CValue.call(this);
  this.agentId_ = agentId;
  this.vec_ = vec;
  this.time_ = time;
  this.val_ = val;
}

// Implements CValue.dtype.
CRegister.prototype.dtype = function() {
  return cvalue.dtypeCRegister;
};

// Decodes the given string into a CRegister.
function decode(s) {
  var x = JSON.parse(s);
  return new CRegister(x.AgentId, x.Vec, x.Time, x.Val);
}

// Implements CValue.applyPatch.
CRegister.prototype.applyPatch = function(isLocal, patch) {
  if (isLocal) {
    this.paused_ = false;
  }
  var other = decode(patch);
  // FIXME: Copy applyPatch logic from cregister.go.
  this.agentId_ = other.agentId_;
  this.vec_ = other.vec_;
  this.time_ = other.time_;
  this.val_ = other.val_;
  this.emit('set', new Set(isLocal, other.val_));
};

// Returns a native JS type that represents this value.
CRegister.prototype.get = function() {
  return this.val_;
};

// Updates this value to the given one, which must be of native JS type.
CRegister.prototype.set = function(value) {
  if (this.paused_) {
    throw new Error('paused');
  }
  this.paused_ = true;
  this.emit('patch', JSON.stringify(value));
};

////////////////////////////////////////////////////////////
// Exports

module.exports = {
  CRegister: CRegister,
  decode: decode,
  Set: Set
};
