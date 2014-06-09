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
