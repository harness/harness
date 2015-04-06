;// Format ANSI to HTML

if(typeof(Drone) === 'undefined') { Drone = {}; }

(function() {
	Drone.LineFormatter = function() {};

	Drone.LineFormatter.prototype = {
		regex: /\u001B\[([0-9]+;?)*[Km]/g,
		styles: [],

		format: function(s) {
			// Check for newline and early exit?
			s = s.replace(/</g, "&lt;");
			s = s.replace(/>/g, "&gt;");

			var output = "";
			var current = 0;
			while (m = this.regex.exec(s)) {
				var part = s.substring(current, m.index);
				current = this.regex.lastIndex;

				var token = s.substr(m.index, this.regex.lastIndex - m.index);
				var code = token.substr(2, token.length-2);

				var pre = "";
				var post = "";

				switch (code) {
					case 'm':
					case '0m':
						var len = this.styles.length;
						for (var i=0; i < len; i++) {
							this.styles.pop();
							post += "</span>"
						}
						break;
					case '30;42m': pre = '<span style="color:black;background:lime">'; break;
					case '36m':
					case '36;1m': pre = '<span style="color:cyan;">'; break;
					case '31m':
					case '31;31m': pre = '<span style="color:red;">'; break;
					case '33m':
					case '33;33m': pre = '<span style="color:yellow;">'; break;
					case '32m':
					case '0;32m': pre = '<span style="color:lime;">'; break;
					case '90m': pre = '<span style="color:gray;">'; break;
					case 'K':
					case '0K':
					case '1K':
					case '2K': break;
				}

				if (pre !== "") {
					this.styles.push(pre);
				}

				output += part + pre + post;
			}

			var part = s.substring(current, s.length);
			output += part;
			return output;
		}
	};
})();
;// Live commit updates

if(typeof(Drone) === 'undefined') { Drone = {}; }

(function () {
	Drone.CommitUpdates = function(socket) {
		if(typeof(socket) === "string") {
			var url = [(window.location.protocol == 'https:' ? 'wss' : 'ws'),
								 '://',
								 window.location.host,
								 socket].join('')
			this.socket = new WebSocket(url);
		} else {
			this.socket = socket;
		}

		this.lineFormatter = new Drone.LineFormatter();
		this.attach();
	}

	Drone.CommitUpdates.prototype = {
		lineBuffer: "",
		autoFollow: false,

		startOutput: function(el) {
			if(typeof(el) === 'string') {
				this.el = document.getElementById(el);
			} else {
				this.el = el;
			}

			if(!this.reqId) {
				this.updateScreen();
			}
		},

		stopOutput: function() {
			this.stoppingRefresh = true;
		},

		attach: function() {
			this.socket.onopen    = this.onOpen;
			this.socket.onerror   = this.onError;
			this.socket.onmessage = this.onMessage.bind(this);
			this.socket.onclose   = this.onClose;
		},

		updateScreen: function() {
			if(this.lineBuffer.length > 0) {
				this.el.innerHTML += this.lineBuffer;
				this.lineBuffer = '';

				if (this.autoFollow) {
					window.scrollTo(0, document.body.scrollHeight);
				}
			}

			if(this.stoppingRefresh) {
				this.stoppingRefresh = false;
			} else {
				window.requestAnimationFrame(this.updateScreen.bind(this));
			}
		},

		onOpen: function() {
			console.log('output websocket open');
		},

		onError: function(e) {
			console.log('websocket error: ' + e);
		},

		onMessage: function(e) {
			this.lineBuffer += this.lineFormatter.format(e.data);
		},

		onClose: function(e) {
			console.log('output websocket closed: ' + JSON.stringify(e));
			window.location.reload();
		}
	};

	// Polyfill rAF for older browsers
	window.requestAnimationFrame = window.requestAnimationFrame ||
		window.webkitRequestAnimationFrame ||
		function(callback, element) {
			return window.setTimeout(function() {
				callback(+new Date());
			}, 1000 / 60);
		};

	window.cancelRequestAnimationFrame = window.cancelRequestAnimationFrame ||
		window.cancelWebkitRequestAnimationFrame ||
		function(fn) {
			window.clearTimeout(fn);
		};

})();