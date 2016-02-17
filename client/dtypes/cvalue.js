// CValue abstract class and related things.

var EventEmitter = require('events').EventEmitter;
var inherits = require('inherits');

////////////////////////////////////////////////////////////
// Events

// Base class for events.
function Event(isLocal) {
  this.isLocal = isLocal;
}

////////////////////////////////////////////////////////////
// CValue

inherits(CValue, EventEmitter);
function CValue() {
  EventEmitter.call(this);
  this.paused_ = false;
}

// Returns this value's dtype.
CValue.prototype.dtype = function() {
  throw new Error('abstract method');
};

// Applies the given encoded patch to this value.
// Similar to ApplyServerPatch in Go.
CValue.prototype.applyPatch = function(isLocal, patch) {
  throw new Error('abstract method');
};

// Returns true iff this value is paused, in which case local mutations are
// disallowed.
CValue.prototype.paused = function() {
  return this.paused_;
};

////////////////////////////////////////////////////////////
// Exports

module.exports = {
  CValue: CValue,
  dtypeCRegister: 'cregister',
  dtypeCString: 'cstring',
  dtypeDelete: 'delete',
  Event: Event
};
