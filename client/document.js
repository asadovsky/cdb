// Document class.

'use strict';

var AsyncModel = require('../editor').AsyncModel;
var logoot = require('./logoot');
var util = require('../util');

module.exports = Document;

function Document(addr, docId, onLoad) {
  var that = this;

  // Initialized by processSnapshotMsg_.
  this.clientId_ = null;
  this.m_ = null;

  // Initialize WebSocket connection.
  var ws = new WebSocket('ws://' + addr);

  ws.onopen = function(e) {
    that.ws_.sendMessage({
      Type: 'Init',
      DocId: docId,
      DataType: 'crdt.Logoot'
    });
  };

  ws.onmessage = function(e) {
    var msg = JSON.parse(e.data);
    switch (msg.Type) {
    case 'Snapshot':
      that.processSnapshotMsg_(msg);
      onLoad(that);
      return;
    case 'Change':
      that.processChangeMsg_(msg);
      return;
    default:
      throw new Error('unknown message type: ' + msg.Type);
    }
  };

  this.ws_ = util.decorateWebSocket(ws);
}

Document.prototype.getModel = function() {
  return this.m_;
};

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

Document.prototype.processSnapshotMsg_ = function(msg) {
  console.assert(this.clientId_ === null);
  this.clientId_ = msg.ClientId;
  this.logoot_ = logoot.decodeLogoot(msg.LogootStr);
  this.m_ = new AsyncModel(this, msg.Text);
};

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
