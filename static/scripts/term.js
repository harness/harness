var Filter, STYLES, defaults, entities, extend, toHexString, _i, _results,
	__slice = [].slice;

// theme stuff:
//   http://ciembor.github.io/4bit
//   https://github.com/lysyi3m/osx-terminal-themes

STYLES = {
	'ef0': 'color:#000',
	'ef1': 'color:#b87a7a',
	'ef2': 'color:#7ab87a',
	'ef3': 'color:#b8b87a',
	'ef4': 'color:#7a7ab8',
	'ef5': 'color:#b87ab8',
	'ef6': 'color:#7ab8b8',
	'ef7': 'color:#d9d9d9',
	'ef8': 'color:#262626',
	'ef9': 'color:#dbbdbd',
	'ef10': 'color:#bddbbd',
	'ef11': 'color:#dbdbbd',
	'ef12': 'color:#bdbddb',
	'ef13': 'color:#dbbddb',
	'ef14': 'color:#bddbdb',
	'ef15': 'color:#FFF',
	'eb0': 'background-color:#000',
	'eb1': 'background-color:#b87a7a',
	'eb2': 'background-color:#7ab87a',
	'eb3': 'background-color:#b8b87a',
	'eb4': 'background-color:#7a7ab8',
	'eb5': 'background-color:#b87ab8',
	'eb6': 'background-color:#7ab8b8',
	'eb7': 'background-color:#d9d9d9',
	'eb8': 'background-color:#262626',
	'eb9': 'background-color:#dbbdbd',
	'eb10': 'background-color:#bddbbd',
	'eb11': 'background-color:#dbdbbd',
	'eb12': 'background-color:#bdbddb',
	'eb13': 'background-color:#dbbddb',
	'eb14': 'background-color:#bddbdb',
	'eb15': 'background-color:#FFF'
};

toHexString = function(num) {
	num = num.toString(16);
	while (num.length < 2) {
		num = "0" + num;
	}
	return num;
};

[0, 1, 2, 3, 4, 5].forEach(function(red) {
	return [0, 1, 2, 3, 4, 5].forEach(function(green) {
		return [0, 1, 2, 3, 4, 5].forEach(function(blue) {
			var b, c, g, n, r, rgb;
			c = 16 + (red * 36) + (green * 6) + blue;
			r = red > 0 ? red * 40 + 55 : 0;
			g = green > 0 ? green * 40 + 55 : 0;
			b = blue > 0 ? blue * 40 + 55 : 0;
			rgb = ((function() {
				var _i, _len, _ref, _results;
				_ref = [r, g, b];
				_results = [];
				for (_i = 0, _len = _ref.length; _i < _len; _i++) {
					n = _ref[_i];
					_results.push(toHexString(n));
				}
				return _results;
			})()).join('');
			STYLES["ef" + c] = "color:#" + rgb;
			return STYLES["eb" + c] = "background-color:#" + rgb;
		});
	});
});

(function() {
	_results = [];
	for (_i = 0; _i <= 23; _i++){ _results.push(_i); }
	return _results;
}).apply(this).forEach(function(gray) {
	var c, l;
	c = gray + 232;
	l = toHexString(gray * 10 + 8);
	STYLES["ef" + c] = "color:#" + l + l + l;
	return STYLES["eb" + c] = "background-color:#" + l + l + l;
});

extend = function() {
	var dest, k, obj, objs, v, _j, _len;
	dest = arguments[0], objs = 2 <= arguments.length ? __slice.call(arguments, 1) : [];
	for (_j = 0, _len = objs.length; _j < _len; _j++) {
		obj = objs[_j];
		for (k in obj) {
			v = obj[k];
			dest[k] = v;
		}
	}
	return dest;
};

defaults = {
	fg: '#FFF',
	bg: '#000',
	newline: false,
	escapeXML: false,
	stream: false
};

Filter = (function() {
	function Filter(options) {
		if (options == null) {
			options = {};
		}
		this.opts = extend({}, defaults, options);
		this.input = [];
		this.stack = [];
		this.stickyStack = [];
	}

	Filter.prototype.toHtml = function(input) {
		var buf;
		this.input = typeof input === 'string' ? [input] : input;
		buf = [];
		this.stickyStack.forEach((function(_this) {
			return function(element) {
				return _this.generateOutput(element.token, element.data, function(chunk) {
					return buf.push(chunk);
				});
			};
		})(this));
		this.forEach(function(chunk) {
			return buf.push(chunk);
		});
		this.input = [];
		return buf.join('');
	};

	Filter.prototype.forEach = function(callback) {
		var buf;
		buf = '';
		this.input.forEach((function(_this) {
			return function(chunk) {
				buf += chunk;
				return _this.tokenize(buf, function(token, data) {
					_this.generateOutput(token, data, callback);
					if (_this.opts.stream) {
						return _this.updateStickyStack(token, data);
					}
				});
			};
		})(this));
		if (this.stack.length) {
			return callback(this.resetStyles());
		}
	};

	Filter.prototype.generateOutput = function(token, data, callback) {
		switch (token) {
			case 'text':
				return callback(this.pushText(data));
			case 'display':
				return this.handleDisplay(data, callback);
			case 'xterm256':
				return callback(this.pushStyle("ef" + data));
		}
	};

	Filter.prototype.updateStickyStack = function(token, data) {
		var notCategory;
		notCategory = function(category) {
			return function(e) {
				return (category === null || e.category !== category) && category !== 'all';
			};
		};
		if (token !== 'text') {
			this.stickyStack = this.stickyStack.filter(notCategory(this.categoryForCode(data)));
			return this.stickyStack.push({
				token: token,
				data: data,
				category: this.categoryForCode(data)
			});
		}
	};

	Filter.prototype.handleDisplay = function(code, callback) {
		code = parseInt(code, 10);
		if (code === -1) {
			callback('<br/>');
		}
		if (code === 0) {
			if (this.stack.length) {
				callback(this.resetStyles());
			}
		}
		if (code === 1) {
			callback(this.pushTag('b'));
		}
		if (code === 2) {

		}
		if ((2 < code && code < 5)) {
			callback(this.pushTag('u'));
		}
		if ((4 < code && code < 7)) {
			callback(this.pushTag('blink'));
		}
		if (code === 7) {

		}
		if (code === 8) {
			callback(this.pushStyle('display:none'));
		}
		if (code === 9) {
			callback(this.pushTag('strike'));
		}
		if (code === 24) {
			callback(this.closeTag('u'));
		}
		if ((29 < code && code < 38)) {
			callback(this.pushStyle("ef" + (code - 30)));
		}
		if (code === 39) {
			callback(this.pushStyle("color:" + this.opts.fg));
		}
		if ((39 < code && code < 48)) {
			callback(this.pushStyle("eb" + (code - 40)));
		}
		if (code === 49) {
			callback(this.pushStyle("background-color:" + this.opts.bg));
		}
		if ((89 < code && code < 98)) {
			callback(this.pushStyle("ef" + (8 + (code - 90))));
		}
		if ((99 < code && code < 108)) {
			return callback(this.pushStyle("eb" + (8 + (code - 100))));
		}
	};

	Filter.prototype.categoryForCode = function(code) {
		code = parseInt(code, 10);
		if (code === 0) {
			return 'all';
		} else if (code === 1) {
			return 'bold';
		} else if ((2 < code && code < 5)) {
			return 'underline';
		} else if ((4 < code && code < 7)) {
			return 'blink';
		} else if (code === 8) {
			return 'hide';
		} else if (code === 9) {
			return 'strike';
		} else if ((29 < code && code < 38) || code === 39 || (89 < code && code < 98)) {
			return 'foreground-color';
		} else if ((39 < code && code < 48) || code === 49 || (99 < code && code < 108)) {
			return 'background-color';
		} else {
			return null;
		}
	};

	Filter.prototype.pushTag = function(tag, style) {
		if (style == null) {
			style = '';
		}
		if (style.length && style.indexOf(':') === -1) {
			style = STYLES[style];
		}
		this.stack.push(tag);
		return ["<" + tag, (style ? " style=\"" + style + "\"" : void 0), ">"].join('');
	};

	Filter.prototype.pushText = function(text) {
		if (this.opts.escapeXML) {
			return entities.encodeXML(text);
		} else {
			return text;
		}
	};

	Filter.prototype.pushStyle = function(style) {
		return this.pushTag("span", style);
	};

	Filter.prototype.closeTag = function(style) {
		var last;
		if (this.stack.slice(-1)[0] === style) {
			last = this.stack.pop();
		}
		if (last != null) {
			return "</" + style + ">";
		}
	};

	Filter.prototype.resetStyles = function() {
		var stack, _ref;
		_ref = [this.stack, []], stack = _ref[0], this.stack = _ref[1];
		return stack.reverse().map(function(tag) {
			return "</" + tag + ">";
		}).join('');
	};

	Filter.prototype.tokenize = function(text, callback) {
		var ansiHandler, ansiMatch, ansiMess, handler, i, length, newline, process, realText, remove, removeXterm256, tokens, _j, _len, _results1;
		ansiMatch = false;
		ansiHandler = 3;
		remove = function(m) {
			return '';
		};
		removeXterm256 = function(m, g1) {
			callback('xterm256', g1);
			return '';
		};
		newline = (function(_this) {
			return function(m) {
				if (_this.opts.newline) {
					callback('display', -1);
				} else {
					callback('text', m);
				}
				return '';
			};
		})(this);
		ansiMess = function(m, g1) {
			var code, _j, _len;
			ansiMatch = true;
			if (g1.trim().length === 0) {
				g1 = '0';
			}
			g1 = g1.trimRight(';').split(';');
			for (_j = 0, _len = g1.length; _j < _len; _j++) {
				code = g1[_j];
				callback('display', code);
			}
			return '';
		};
		realText = function(m) {
			callback('text', m);
			return '';
		};
		tokens = [
			{
				pattern: /^\x08+/,
				sub: remove
			}, {
				pattern: /^\x1b\[[012]?K/,
				sub: remove
			}, {
				pattern: /^\x1b\[38;5;(\d+)m/,
				sub: removeXterm256
			}, {
				pattern: /^\n+/,
				sub: newline
			}, {
				pattern: /^\r+/,
				sub: newline
			}, {
				pattern: /^\x1b\[((?:\d{1,3};?)+|)m/,
				sub: ansiMess
			}, {
				pattern: /^\x1b\[?[\d;]{0,3}/,
				sub: remove
			}, {
				pattern: /^([^\x1b\x08\n]+)/,
				sub: realText
			}
		];
		process = function(handler, i) {
			var matches;
			if (i > ansiHandler && ansiMatch) {
				return;
			} else {
				ansiMatch = false;
			}
			matches = text.match(handler.pattern);
			text = text.replace(handler.pattern, handler.sub);
			if (matches == null) {

			}
		};
		_results1 = [];
		while ((length = text.length) > 0) {
			for (i = _j = 0, _len = tokens.length; _j < _len; i = ++_j) {
				handler = tokens[i];
				process(handler, i);
			}
			if (text.length === length) {
				break;
			} else {
				_results1.push(void 0);
			}
		}
		return _results1;
	};

	return Filter;

})();