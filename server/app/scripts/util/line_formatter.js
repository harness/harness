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
