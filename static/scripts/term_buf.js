;// Live commit updates

if(typeof(Drone) === 'undefined') { Drone = {}; }

(function () {
	Drone.Buffer = function() {
		this.lineFormatter = new Filter({stream: true, newline: false});
	}

	Drone.Buffer.prototype = {
		lineBuffer: "",
		autoFollow: false,
		stoppingRefresh: false,

		start: function(el) {
			if(typeof(el) === 'string') {
				this.el = document.getElementById(el);
			} else {
				this.el = el;
			}

			this.el.innerHTML="";
			this.update();
		},

		stop: function() {
			this.stoppingRefresh = true;
		},

		update: function() {
			if(this.lineBuffer.length > 0) {
				this.el.innerHTML += this.lineFormatter.toHtml(escapeHTML(this.lineBuffer));
				this.lineBuffer = '';

				if (this.autoFollow) {
					window.scrollTo(0, document.body.scrollHeight);
				}
			}

			if(this.stoppingRefresh) {
				this.stoppingRefresh = false;
			} else {
				window.requestAnimationFrame(this.update.bind(this));
			}
		},

		write: function(data) {
			this.lineBuffer += data;
		}
	};

})();