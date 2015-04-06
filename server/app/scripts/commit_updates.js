;// Live commit updates

if(typeof(Drone) === 'undefined') { Drone = {}; }

(function () {
	Drone.Console = function() {
		this.lineFormatter = new Drone.LineFormatter();
	}

	Drone.Console.prototype = {
		lineBuffer: "",
		autoFollow: false,

		start: function(el) {
			if(typeof(el) === 'string') {
				this.el = document.getElementById(el);
			} else {
				this.el = el;
			}

			this.update();
		},

		stop: function() {
			this.stoppingRefresh = true;
		},

		update: function() {
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

		write: function(e) {
			this.lineBuffer += this.lineFormatter.format(e.data);
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
