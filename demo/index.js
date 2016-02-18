/* jshint newcap: false */

var _ = require('lodash');
var eddie = require('eddie');
var React = require('react'), h = require('react-h-function')(React);
var ReactDOM = require('react-dom');
var url = require('url');

var Store = require('../client/store');

function newEditor(el, type, model) {
  if (type === 'eddie') {
    return new eddie.EddieEditor(el, model);
  } else {
    console.assert(type === 'textarea');
    return new eddie.TextareaEditor(el, model);
  }
}

var Editor = React.createFactory(React.createClass({
  displayName: 'Editor',
  componentDidMount: function() {
    var that = this, el = ReactDOM.findDOMNode(this);
    var st = new Store(this.props.addr);
    st.open(function() {
      var model = st.getOrCreate('0', 'cstring');
      var ed = newEditor(el, that.props.type, model);
      if (that.props.focus) ed.focus();
    });
  },
  render: function() {
    return h('div');
  }
}));

var Page = React.createFactory(React.createClass({
  displayName: 'Page',
  render: function() {
    var props = _.pick(this.props, ['type', 'addr']);
    return h('div', [
      h('pre', JSON.stringify(props, null, 2)),
      h('div', [
        Editor(_.assign({focus: true}, props)), h('br'), Editor(props)
      ])
    ]);
  }
}));

var u = url.parse(window.location.href, true);

ReactDOM.render(Page({
  mode: u.query.mode || 'local',
  type: u.query.type || 'eddie',
  addr: u.query.addr || 'localhost:4000'
}), document.getElementById('page'));
