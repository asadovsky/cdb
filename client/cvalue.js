// CValue abstract class.

var EventEmitter = require('events').EventEmitter;
var inherits = require('inherits');

inherits(CValue, EventEmitter);
module.exports = CValue;

function CValue() {
  throw new Error('FIXME');
}

// Returns a native JS type that represents this value.
CValue.prototype.value = function() {
  throw new Error('abstract method');
};

// FIXME: Move the stuff below to Collection and (mostly) to CString.

////////////////////////////////////////
// Model event handlers

Document.prototype.handleReplaceText = function(pos, len, value) {
  var ops = new Array(len);
  for (var i = 0; i < len; i++) {
    ops[i] = new logoot.Delete(this.logoot_.pid(pos + i));
  }
  if (value) {
    var prevPid = pos === 0 ? '' : this.logoot_.pid(pos - 1);
    var nextPid = '';
    if (pos + len < this.logoot_.len()) {
      nextPid = this.logoot_.pid(pos + len);
    }
    ops.push(new logoot.ClientInsert(prevPid, nextPid, value));
  }
  this.sendOps_(ops);
};

////////////////////////////////////////
// Incoming message handlers

Document.prototype.processChangeMsg_ = function(msg) {
  var that = this;
  var isLocal = msg.ClientId === this.clientId_;

  // Apply all mutations, regardless of whether they originated from this client
  // (i.e. unidirectional data flow).
  var ops = logoot.decodeOps(msg.OpStrs);

  // Consecutive single-char insertions and deletions are common, and applying
  // lots of point mutations to the model is expensive (e.g. applying 400 point
  // deletions takes hundreds of milliseconds), so we compact such ops when
  // updating the model.
  var pos = -1, len = 0, value = '';
  function applyReplaceText() {
    if (pos !== -1) {
      that.m_.applyReplaceText(isLocal, pos, len, value);
    }
  }

  for (var i = 0; i < ops.length; i++) {
    var op = ops[i];
    switch(op.constructor.name) {
    case 'Insert':
      var insertPos = this.logoot_.applyInsertText(op);
      if (insertPos === pos + value.length) {
        value += op.value;
      } else {
        applyReplaceText();
        pos = insertPos;
        len = 0;
        value = op.value;
      }
      break;
    case 'Delete':
      var deletePos = this.logoot_.applyDeleteText(op);
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

////////////////////////////////////////
// Other private helpers

// TODO: Delta encoding; more efficient pid encoding; compression.
Document.prototype.sendOps_ = function(ops) {
  if (!ops.length) {
    return;
  }
  this.ws_.sendMessage({
    Type: 'Update',
    ClientId: this.clientId_,
    OpStrs: logoot.encodeOps(ops)
  });
};
