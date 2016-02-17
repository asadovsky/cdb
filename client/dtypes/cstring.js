// CString class.
// Mostly mirrors server/dtypes/cstring/cstring.go.

var _ = require('lodash');
var inherits = require('inherits');

var cvalue = require('./cvalue');
var lib = require('../lib');

////////////////////////////////////////////////////////////
// Events

inherits(ReplaceText, cvalue.Event);
function ReplaceText(isLocal, pos, len, value) {
  cvalue.Event.call(this, isLocal);
  this.pos = pos;
  this.len = len;
  this.value = value;
}

inherits(SetSelectionRange, cvalue.Event);
function SetSelectionRange(isLocal, start, end) {
  cvalue.Event.call(this, isLocal);
  this.start = start;
  this.end = end;
}

////////////////////////////////////////////////////////////
// CString

function Id(pos, agentId) {
  this.pos = pos;
  this.agentId = agentId;
}

function Pid(ids, seq) {
  this.ids = ids;
  this.seq = seq;
}

Pid.prototype.less = function(other) {
  for (var i = 0; i < this.ids.length; i++) {
    if (i === other.ids.length) {
      return false;
    }
    var v = this.ids[i], vo = other.ids[i];
    if (v.pos !== vo.pos) {
      return v.pos < vo.pos;
    } else if (v.agentId !== vo.agentId) {
      return v.agentId < vo.agentId;
    }
  }
  if (this.ids.length === other.ids.length) {
    return this.seq < other.seq;
  }
  return true;
};

Pid.prototype.encode = function() {
  return _.map(this.ids, function(id) {
    return [id.pos, id.agentId].join('.');
  }).join(':') + '~' + this.seq;
};

function decodePid(s) {
  var idsAndSeq = s.split('~');
  if (idsAndSeq.length !== 2 ) {
    throw new Error('invalid pid: ' + s);
  }
  var seq = Number(idsAndSeq[1]);
  var ids = _.map(s.split(':'), function(idStr) {
    var parts = idStr.split('.');
    if (parts.length !== 2) {
      throw new Error('invalid id: ' + idStr);
    }
    return new Id(Number(parts[0]), Number(parts[1]));
  });
  return new Pid(ids, seq);
}

function Op() {}

Op.prototype.encode = function() {
  throw new Error('abstract method');
};

inherits(ClientInsert, Op);
function ClientInsert(prevPid, nextPid, value) {
  Op.call(this);
  this.prevPid = prevPid;
  this.nextPid = nextPid;
  this.value = value;
}

ClientInsert.prototype.encode = function() {
  var prevPid = this.prevPid ? this.prevPid.encode() : '';
  var nextPid = this.nextPid ? this.nextPid.encode() : '';
  return ['ci', prevPid, nextPid, this.value].join(',');
};

inherits(Insert, Op);
function Insert(pid, value) {
  Op.call(this);
  this.pid = pid;
  this.value = value;
}

Insert.prototype.encode = function() {
  return ['i', this.pid.encode(), this.value].join(',');
};

inherits(Delete, Op);
function Delete(pid) {
  Op.call(this);
  this.pid = pid;
}

Delete.prototype.encode = function() {
  return ['d', this.pid.encode()].join(',');
};

function newParseError(s) {
  return new Error('failed to parse op: ' + s);
}

function decodeOp(s) {
  var parts;
  var t = s.split(',', 1)[0];
  switch (t) {
  case 'ci':
    parts = lib.splitN(s, ',', 4);
    if (parts.length < 4) {
      throw newParseError(s);
    }
    return new ClientInsert(decodePid(parts[1]), decodePid(parts[2]), parts[3]);
  case 'i':
    parts = lib.splitN(s, ',', 3);
    if (parts.length < 3) {
      throw newParseError(s);
    }
    return new Insert(decodePid(parts[1]), parts[2]);
  case 'd':
    parts = lib.splitN(s, ',', 2);
    if (parts.length < 2) {
      throw newParseError(s);
    }
    return new Delete(decodePid(parts[1]));
  default:
    throw new Error('unknown op type: ' + t);
  }
}

function encodePatch(ops) {
  var strs = new Array(ops.length);
  for (var i = 0; i < ops.length; i++) {
    strs[i] = ops[i].encode();
  }
  return JSON.stringify(strs);
}

function decodePatch(s) {
  var strs = JSON.parse(s);
  var ops = new Array(strs.length);
  for (var i = 0; i < strs.length; i++) {
    ops[i] = decodeOp(strs[i]);
  }
  return ops;
}

function Atom(pid, value) {
  this.pid = pid;
  this.value = value;
}

inherits(CString, cvalue.CValue);
function CString(atoms) {
  cvalue.CValue.call(this);
  this.atoms_ = atoms;
  this.text_ = _.map(atoms, 'value').join('');
  this.selStart_ = 0;
  this.selEnd_ = 0;
}

// Implements CValue.dtype.
CString.prototype.dtype = function() {
  return cvalue.dtypeCString;
};

// Decodes the given string into a CString.
function decode(s) {
  var atoms = JSON.parse(s);
  return new CString(_.map(atoms, function(atom) {
    return new Atom(decodePid(atom.Pid), atom.Value);
  }));
}

// Implements CValue.applyPatch.
CString.prototype.applyPatch = function(isLocal, patch) {
  var that = this;
  if (isLocal) {
    this.paused_ = false;
  }

  // Consecutive single-char insertions and deletions are common, and applying
  // lots of point mutations to this.text_ is expensive (e.g. applying 400 point
  // deletions takes hundreds of milliseconds), so we compact such ops when
  // updating this.text_.
  // TODO: Use a rope data structure, e.g. the jumprope npm package.
  var pos = -1, len = 0, value = '';
  function applyReplaceText() {
    if (pos !== -1) {
      that.applyReplaceText_(isLocal, pos, len, value);
    }
  }

  var ops = decodePatch(patch);
  for (var i = 0; i < ops.length; i++) {
    var op = ops[i];
    switch(op.constructor.name) {
    case 'insert':
      var insertPos = this.search_(op.pid);
      this.atoms_.splice(insertPos, 0, new Atom(op.pid, op.value));
      if (insertPos === pos + value.length) {
        value += op.value;
      } else {
        applyReplaceText();
        pos = insertPos;
        len = 0;
        value = op.value;
      }
      break;
    case 'delete':
      var deletePos = this.search_(op.pid);
      this.atoms_.splice(deletePos, 1);
      if (deletePos === pos) {
        len++;
      } else {
        applyReplaceText();
        pos = deletePos;
        len = 1;
        value = '';
      }
      break;
    default:
      throw new Error(op.constructor.name);
    }
  }
  applyReplaceText();
};

// Returns the text, a string.
CString.prototype.getText = function() {
  return this.text_;
};

// Returns the selection range, an array representing [start, end).
CString.prototype.getSelectionRange = function(value) {
  return [this.selStart_, this.selEnd_];
};

// Replaces 'len' characters, starting at position 'pos', with 'value'.
// Assumes line breaks have been canonicalized to \n.
CString.prototype.replaceText = function(pos, len, value) {
  if (this.paused_) {
    throw new Error('paused');
  }
  if (len === 0 && value.length === 0) {
    return;
  }
  this.paused_ = true;
  var ops = new Array(len);
  for (var i = 0; i < len; i++) {
    ops[i] = new Delete(this.atoms_[pos + i].pid);
  }
  if (value) {
    var prevPid = pos === 0 ? '' : this.atoms_[pos - 1].pid;
    var nextPid = '';
    if (pos + len < this.atoms_.length) {
      nextPid = this.atoms_[pos + len].pid;
    }
    ops.push(new ClientInsert(prevPid, nextPid, value));
  }
  this.emit('patch', encodePatch(ops));
};

// Sets the selection range to [start, end).
CString.prototype.setSelectionRange = function(start, end) {
  if (this.paused_) {
    throw new Error('paused');
  }
  if (this.selStart_ === start && this.selEnd_ === end) {
    return;
  }
  // TODO: Set this.paused_ and notify server. For now, we simply update local
  // state and emit a 'setSelectionRange' event.
  this.selStart_ = start;
  this.selEnd_ = end;
  this.emit('setSelectionRange', new SetSelectionRange(true, start, end));
};

CString.prototype.search_ = function(pid) {
  var that = this;
  return lib.search(this.atoms_.length, function(i) {
    return !that.atoms_[i].pid.less(pid);
  });
};

// Note: A single call to CString.replaceText can result in multiple calls to
// CString.applyReplaceText_.
CString.prototype.applyReplaceText_ = function(isLocal, pos, len, value) {
  if (len === 0 && value.length === 0) {
    return;
  }
  var t = this.text_;
  if (pos < 0 || pos + len > t.length) {
    throw new Error('out of bounds');
  }
  this.text_ = t.substr(0, pos) + value + t.substr(pos + len);
  // Update selection range.
  if (isLocal) {
    this.selStart_ = pos + value.length;
    this.selEnd_ = this.selStart_;
  } else {
    if (this.selStart_ >= pos) {
      this.selStart_ = Math.max(pos, this.selStart_ - len) + value.length;
    }
    if (this.selEnd_ >= pos) {
      this.selEnd_ = Math.max(pos, this.selEnd_ - len) + value.length;
    }
  }
  this.emit('replaceText', new ReplaceText(isLocal, pos, len, value));
};

////////////////////////////////////////////////////////////
// Exports

module.exports = {
  CString: CString,
  decode: decode,
  ReplaceText: ReplaceText,
  SetSelectionRange: SetSelectionRange
};
