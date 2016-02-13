// CRegister class.
// Mostly mirrors server/dtypes/cregister/cregister.go.

// FIXME: Add events:
// - Set: {isLocal, value}

var inherits = require('inherits');

var CValue = require('./cvalue');

inherits(CRegister, CValue);
module.exports = CRegister;

// FIXME: Call parent constructor, here and elsewhere.
function CRegister() {
  throw new Error('FIXME');
}

// Returns a native JS type that represents this value.
CRegister.prototype.get = function() {
  throw new Error('FIXME');
};

// Updates this value to the given one, which must be of native JS type.
CRegister.prototype.set = function(value) {
  throw new Error('FIXME');
};
