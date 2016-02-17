// Helper functions.

var cregister = require('./cregister');
var cstring = require('./cstring');
var cvalue = require('./cvalue');

// Decodes the given value.
exports.decodeValue = function(dtype, value) {
  switch (dtype) {
  case cvalue.dtypeCRegister:
    return cregister.decode(value);
  case cvalue.dtypeCString:
    return cstring.decode(value);
  default:
    throw new Error('unknown dtype: ' + dtype);
  }
};

// Returns a new zero value of the given dtype.
exports.newZeroValue = function(dtype) {
  switch (dtype) {
  case cvalue.dtypeCRegister:
    return new cregister.CRegister(undefined);
  case cvalue.dtypeCString:
    return new cstring.CString('');
  default:
    throw new Error('unknown dtype: ' + dtype);
  }
};
