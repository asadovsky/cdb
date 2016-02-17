// Conn class, representing a JSON message pipe.

var EventEmitter = require('events').EventEmitter;
var inherits = require('inherits');

inherits(Conn, EventEmitter);
module.exports = Conn;

function Conn(addr) {
  EventEmitter.call(this);
  var that = this;
  this.ws_ = new WebSocket('ws://' + addr);

  this.ws_.onopen = function(e) {
    if (process.env.DEBUG_SOCKET) {
      console.log('socket.open');
    }
    that.emit('open');
  };

  this.ws_.onclose = function(e) {
    if (process.env.DEBUG_SOCKET) {
      console.log('socket.close');
    }
    that.emit('close');
  };

  this.ws_.onmessage = function(e) {
    if (process.env.DEBUG_SOCKET) {
      console.log('socket.recv: ' + e.data);
    }
    that.emit('recv', JSON.parse(e.data));
  };
}

Conn.prototype.send = function(msg) {
  var that = this;
  var json = JSON.stringify(msg);
  if (process.env.DEBUG_SOCKET) {
    console.log('socket.send: ' + json);
  }
  function send() {
    that.ws_.send(json);
  }
  if (process.env.DEBUG_DELAY) {
    window.setTimeout(send, Number(process.env.DEBUG_DELAY));
  } else {
    send();
  }
};
