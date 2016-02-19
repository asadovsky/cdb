// Utility functions.

exports.atoi = function(s) {
  var n = Number(s);
  if (s === '' || isNaN(n)) {
    throw new Error('not a number: ' + s);
  }
  return n;
};

exports.setImmediate = function(cb) {
  window.setTimeout(cb, 0);
};

// Mimics Go's strings.SplitN.
exports.splitN = function(s, sep, n) {
  var parts = s.split(sep);
  if (parts.length >= n) {
    parts[n - 1] = parts.slice(n - 1).join(',');
    parts = parts.slice(0, 3);
  }
  return parts;
};

// Binary search. Mimics Go's sort.Search.
// Returns the smallest index i in [0, n) at which f(i) is true, assuming that
// on the range [0, n), f(i) == true implies f(i+1) == true. If there is no such
// index, returns n. Calls f(i) only for i in the range [0, n).
exports.search = function(n, f) {
  var i = 0, j = n;
  while (i < j) {
    var h = i + Math.floor((j-i)/2);
    if (!f(h)) {
      i = h + 1;
    } else {
      j = h;
    }
  }
  return i;
};
