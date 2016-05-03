;// Live commit updates

if(typeof(Drone) === 'undefined') { Drone = {}; }

(function () {
	Drone.Buffer = function() {
		this.filter = new Filter();
		// this.lineFormatter = {
		// 	toHtml: function(text) {
		// 		return this.filter.append(text);
		// 	}
		// };
	};

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
				this.filter.append(this.el, escapeHTML(this.lineBuffer));
				this.lineBuffer = '';
				var folders = this.el.getElementsByClassName('fold');
				folders = [].slice.call(folders);

				if (folders.length) {
					folders.pop();
					folders.forEach(function(folder) {
						if (!folder.classList.contains('closed')) {
							folder.classList.add('closed');
						}
					});
				}

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

		write: function(event) {
			// console.log(event);
			var data = event;
			if (event.data !== undefined) {
				data = event.data;
			}
			this.lineBuffer += data;
		}
	};

})();