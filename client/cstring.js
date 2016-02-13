// CString class.
// Mostly mirrors server/dtypes/cstring/cstring.go.

// FIXME: Add events:
// - ReplaceText: {isLocal, pos, len, value}
// - SetSelectionRange: {isLocal, start, end}

var _ = require('lodash');
var inherits = require('inherits');

var CValue = require('./cvalue');
var util = require('./util');

inherits(CString, CValue);
module.exports = CString;

function pidLess(a, b) {
  for (var i = 0; i < a.Ids.length; i++) {
    if (i === b.Ids.length) {
      return false;
    }
    var va = a.Ids[i], vb = b.Ids[i];
    if (va.Pos !== vb.Pos) {
      return va.Pos < vb.Pos;
    } else if (va.AgentId !== vb.AgentId) {
      return va.AgentId < vb.AgentId;
    }
  }
  return a.Ids.length < b.Ids.length;
}

function pidEncode(pid) {
  return _.map(pid.Ids, function(id) {
    return [id.Pos, id.AgentId].join('.');
  }).join(':');
}

function decodePid(s) {
  return {Ids: _.map(s.split(':'), function(idStr) {
    var parts = idStr.split('.');
    if (parts.length !== 2) {
      throw new Error('invalid pid: ' + s);
    }
    return {Pos: Number(parts[0]), AgentId: Number(parts[1])};
  })};
}

function Op() {}

Op.prototype.encode = function() {
  throw new Error('not implemented');
};

inherits(ClientInsert, Op);
function ClientInsert(prevPid, nextPid, value) {
  this.prevPid = prevPid;
  this.nextPid = nextPid;
  this.value = value;
}

ClientInsert.prototype.encode = function() {
  var prevPid = this.prevPid ? pidEncode(this.prevPid) : '';
  var nextPid = this.nextPid ? pidEncode(this.nextPid) : '';
  return ['ci', prevPid, nextPid, this.value].join(',');
};

inherits(Insert, Op);
function Insert(pid, value) {
  this.pid = pid;
  this.value = value;
}

Insert.prototype.encode = function() {
  return ['i', pidEncode(this.pid), this.value].join(',');
};

inherits(Delete, Op);
function Delete(pid) {
  this.pid = pid;
}

Delete.prototype.encode = function() {
  return ['d', pidEncode(this.pid)].join(',');
};

function newParseError(s) {
  return new Error('failed to parse op: ' + s);
}

function decodeOp(s) {
  var parts;
  var t = s.split(',', 1)[0];
  switch (t) {
  case 'ci':
    parts = util.splitN(s, ',', 4);
    if (parts.length < 4) {
      throw newParseError(s);
    }
    return new ClientInsert(decodePid(parts[1]), decodePid(parts[2]), parts[3]);
  case 'i':
    parts = util.splitN(s, ',', 3);
    if (parts.length < 3) {
      throw newParseError(s);
    }
    return new Insert(decodePid(parts[1]), parts[2]);
  case 'd':
    parts = util.splitN(s, ',', 2);
    if (parts.length < 2) {
      throw newParseError(s);
    }
    return new Delete(decodePid(parts[1]));
  default:
    throw new Error('unknown op type: ' + t);
  }
}

function encodeOps(ops) {
  var strs = new Array(ops.length);
  for (var i = 0; i < ops.length; i++) {
    strs[i] = ops[i].encode();
  }
  return strs;
}

function decodeOps(strs) {
  var ops = new Array(strs.length);
  for (var i = 0; i < strs.length; i++) {
    ops[i] = decodeOp(strs[i]);
  }
  return ops;
}

function Logoot(atoms) {
  this.atoms_ = atoms;
}

function decodeLogoot(s) {
  return new Logoot(JSON.parse(s));
}

Logoot.prototype.len = function() {
  return this.atoms_.length;
};

Logoot.prototype.pid = function(i) {
  return this.atoms_[i].Pid;
};

Logoot.prototype.applyInsertText = function(op) {
  var p = this.search_(op.pid);
  this.atoms_.splice(p, 0, {Pid: op.pid, Value: op.value});
  return p;
};

Logoot.prototype.applyDeleteText = function(op) {
  var p = this.search_(op.pid);
  this.atoms_.splice(p, 1);
  return p;
};

Logoot.prototype.search_ = function(pid) {
  var that = this;
  return util.search(this.atoms_.length, function(i) {
    return !pidLess(that.atoms_[i].Pid, pid);
  });
};

function CString() {
  throw new Error('FIXME');
}

// Returns the text, a string.
CString.prototype.getText = function() {
  throw new Error('FIXME');
};

// Returns the selection range, an array representing [start, end).
CString.prototype.getSelectionRange = function(value) {
  throw new Error('FIXME');
};

// Replaces 'len' characters, starting at position 'pos', with 'value'.
CString.prototype.replaceText = function(pos, len, value) {
  throw new Error('FIXME');
};

// Sets the selection range to [start, end).
CString.prototype.setSelectionRange = function(start, end) {
  throw new Error('FIXME');
};
