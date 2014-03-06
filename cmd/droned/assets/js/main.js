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
$(function() {
  var projectForm = $(".form-repo")
  var repoOwnerField = projectForm.find("input[name=owner]")
  var repoNameField = projectForm.find("input[name=name]")

  var projectsBlock = $(".github-projects")
  var projectsList = projectsBlock.find(".projects-list")
  var spinnerBlock = projectsBlock.find(".spinner")

  spinner = new Spinner({
    lines: 12,
    speed: 0.5,
    width: 4,
    length: 10
  })
  spinner.spin(spinnerBlock[0])

  GithubRepos.get(function(response) {
    $.each(response, function(i, repo) {
      var title = repo.owner + "/" + repo.name

      item = $("<div></div>").addClass("item")
      link = $("<a></a>").text(title).attr("href", "#")
      if(repo.private) {
        icon = $("<span></span>").addClass("glyphicon").addClass("glyphicon-lock")
        item.append("&nbsp;")
        item.append(icon)
      }

      item.append(link)

      item.data('owner', repo.owner)
      item.data('name', repo.name)

      projectsList.append(item)
    })

    spinner.stop()
    spinnerBlock.remove()
  })

  projectsList.on('click', 'a', function(event) {
    var badge = $(event.target).parent()

    repoOwnerField.val(badge.data('owner'))
    repoNameField.val(badge.data('name'))
    repoNameField.focus()

    $(document).scrollTop(projectForm.offset().top)

    return false
  })

  projectForm.on('submit', function() {
    $("#successAlert").hide();
    $("#failureAlert").hide();
    $('#submitButton').button('loading');

    $.ajax({
      type: "POST",
      url: projectForm.attr("target"),
      data: projectForm.serialize(),
      success: function(response, status) {
        var name = $("input[name=name]").val()
        var owner = $("input[name=owner]").val()
        var domain = $("input[name=domain]").val()
        window.location.pathname = "/" + domain + "/"+owner+"/"+name
      },
      error: function() {
        $("#failureAlert").text("Unable to setup the Repository");
        $("#failureAlert").show().removeClass("hide");
        $('#submitButton').button('reset');
      }
    });

    return false;
  })
})
GithubRepos = {
  url: "/new/github.com/available_repos",
  get: function(success) {
    $.getJSON(this.url, function(response) {
      success(response)
    })
  }
}
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
/**
 * Copyright (c) 2011-2013 Felix Gnass
 * Licensed under the MIT license
 */
(function(root, factory) {

  /* CommonJS */
  if (typeof exports == 'object')  module.exports = factory()

  /* AMD module */
  else if (typeof define == 'function' && define.amd) define(factory)

  /* Browser global */
  else root.Spinner = factory()
}
(this, function() {
  "use strict";

  var prefixes = ['webkit', 'Moz', 'ms', 'O'] /* Vendor prefixes */
    , animations = {} /* Animation rules keyed by their name */
    , useCssAnimations /* Whether to use CSS animations or setTimeout */

  /**
   * Utility function to create elements. If no tag name is given,
   * a DIV is created. Optionally properties can be passed.
   */
  function createEl(tag, prop) {
    var el = document.createElement(tag || 'div')
      , n

    for(n in prop) el[n] = prop[n]
    return el
  }

  /**
   * Appends children and returns the parent.
   */
  function ins(parent /* child1, child2, ...*/) {
    for (var i=1, n=arguments.length; i<n; i++)
      parent.appendChild(arguments[i])

    return parent
  }

  /**
   * Insert a new stylesheet to hold the @keyframe or VML rules.
   */
  var sheet = (function() {
    var el = createEl('style', {type : 'text/css'})
    ins(document.getElementsByTagName('head')[0], el)
    return el.sheet || el.styleSheet
  }())

  /**
   * Creates an opacity keyframe animation rule and returns its name.
   * Since most mobile Webkits have timing issues with animation-delay,
   * we create separate rules for each line/segment.
   */
  function addAnimation(alpha, trail, i, lines) {
    var name = ['opacity', trail, ~~(alpha*100), i, lines].join('-')
      , start = 0.01 + i/lines * 100
      , z = Math.max(1 - (1-alpha) / trail * (100-start), alpha)
      , prefix = useCssAnimations.substring(0, useCssAnimations.indexOf('Animation')).toLowerCase()
      , pre = prefix && '-' + prefix + '-' || ''

    if (!animations[name]) {
      sheet.insertRule(
        '@' + pre + 'keyframes ' + name + '{' +
        '0%{opacity:' + z + '}' +
        start + '%{opacity:' + alpha + '}' +
        (start+0.01) + '%{opacity:1}' +
        (start+trail) % 100 + '%{opacity:' + alpha + '}' +
        '100%{opacity:' + z + '}' +
        '}', sheet.cssRules.length)

      animations[name] = 1
    }

    return name
  }

  /**
   * Tries various vendor prefixes and returns the first supported property.
   */
  function vendor(el, prop) {
    var s = el.style
      , pp
      , i

    prop = prop.charAt(0).toUpperCase() + prop.slice(1)
    for(i=0; i<prefixes.length; i++) {
      pp = prefixes[i]+prop
      if(s[pp] !== undefined) return pp
    }
    if(s[prop] !== undefined) return prop
  }

  /**
   * Sets multiple style properties at once.
   */
  function css(el, prop) {
    for (var n in prop)
      el.style[vendor(el, n)||n] = prop[n]

    return el
  }

  /**
   * Fills in default values.
   */
  function merge(obj) {
    for (var i=1; i < arguments.length; i++) {
      var def = arguments[i]
      for (var n in def)
        if (obj[n] === undefined) obj[n] = def[n]
    }
    return obj
  }

  /**
   * Returns the absolute page-offset of the given element.
   */
  function pos(el) {
    var o = { x:el.offsetLeft, y:el.offsetTop }
    while((el = el.offsetParent))
      o.x+=el.offsetLeft, o.y+=el.offsetTop

    return o
  }

  /**
   * Returns the line color from the given string or array.
   */
  function getColor(color, idx) {
    return typeof color == 'string' ? color : color[idx % color.length]
  }

  // Built-in defaults

  var defaults = {
    lines: 12,            // The number of lines to draw
    length: 7,            // The length of each line
    width: 5,             // The line thickness
    radius: 10,           // The radius of the inner circle
    rotate: 0,            // Rotation offset
    corners: 1,           // Roundness (0..1)
    color: '#000',        // #rgb or #rrggbb
    direction: 1,         // 1: clockwise, -1: counterclockwise
    speed: 1,             // Rounds per second
    trail: 100,           // Afterglow percentage
    opacity: 1/4,         // Opacity of the lines
    fps: 20,              // Frames per second when using setTimeout()
    zIndex: 2e9,          // Use a high z-index by default
    className: 'spinner', // CSS class to assign to the element
    top: 'auto',          // center vertically
    left: 'auto',         // center horizontally
    position: 'relative'  // element position
  }

  /** The constructor */
  function Spinner(o) {
    if (typeof this == 'undefined') return new Spinner(o)
    this.opts = merge(o || {}, Spinner.defaults, defaults)
  }

  // Global defaults that override the built-ins:
  Spinner.defaults = {}

  merge(Spinner.prototype, {

    /**
     * Adds the spinner to the given target element. If this instance is already
     * spinning, it is automatically removed from its previous target b calling
     * stop() internally.
     */
    spin: function(target) {
      this.stop()

      var self = this
        , o = self.opts
        , el = self.el = css(createEl(0, {className: o.className}), {position: o.position, width: 0, zIndex: o.zIndex})
        , mid = o.radius+o.length+o.width
        , ep // element position
        , tp // target position

      if (target) {
        target.insertBefore(el, target.firstChild||null)
        tp = pos(target)
        ep = pos(el)
        css(el, {
          left: (o.left == 'auto' ? tp.x-ep.x + (target.offsetWidth >> 1) : parseInt(o.left, 10) + mid) + 'px',
          top: (o.top == 'auto' ? tp.y-ep.y + (target.offsetHeight >> 1) : parseInt(o.top, 10) + mid)  + 'px'
        })
      }

      el.setAttribute('role', 'progressbar')
      self.lines(el, self.opts)

      if (!useCssAnimations) {
        // No CSS animation support, use setTimeout() instead
        var i = 0
          , start = (o.lines - 1) * (1 - o.direction) / 2
          , alpha
          , fps = o.fps
          , f = fps/o.speed
          , ostep = (1-o.opacity) / (f*o.trail / 100)
          , astep = f/o.lines

        ;(function anim() {
          i++;
          for (var j = 0; j < o.lines; j++) {
            alpha = Math.max(1 - (i + (o.lines - j) * astep) % f * ostep, o.opacity)

            self.opacity(el, j * o.direction + start, alpha, o)
          }
          self.timeout = self.el && setTimeout(anim, ~~(1000/fps))
        })()
      }
      return self
    },

    /**
     * Stops and removes the Spinner.
     */
    stop: function() {
      var el = this.el
      if (el) {
        clearTimeout(this.timeout)
        if (el.parentNode) el.parentNode.removeChild(el)
        this.el = undefined
      }
      return this
    },

    /**
     * Internal method that draws the individual lines. Will be overwritten
     * in VML fallback mode below.
     */
    lines: function(el, o) {
      var i = 0
        , start = (o.lines - 1) * (1 - o.direction) / 2
        , seg

      function fill(color, shadow) {
        return css(createEl(), {
          position: 'absolute',
          width: (o.length+o.width) + 'px',
          height: o.width + 'px',
          background: color,
          boxShadow: shadow,
          transformOrigin: 'left',
          transform: 'rotate(' + ~~(360/o.lines*i+o.rotate) + 'deg) translate(' + o.radius+'px' +',0)',
          borderRadius: (o.corners * o.width>>1) + 'px'
        })
      }

      for (; i < o.lines; i++) {
        seg = css(createEl(), {
          position: 'absolute',
          top: 1+~(o.width/2) + 'px',
          transform: o.hwaccel ? 'translate3d(0,0,0)' : '',
          opacity: o.opacity,
          animation: useCssAnimations && addAnimation(o.opacity, o.trail, start + i * o.direction, o.lines) + ' ' + 1/o.speed + 's linear infinite'
        })

        if (o.shadow) ins(seg, css(fill('#000', '0 0 4px ' + '#000'), {top: 2+'px'}))
        ins(el, ins(seg, fill(getColor(o.color, i), '0 0 1px rgba(0,0,0,.1)')))
      }
      return el
    },

    /**
     * Internal method that adjusts the opacity of a single line.
     * Will be overwritten in VML fallback mode below.
     */
    opacity: function(el, i, val) {
      if (i < el.childNodes.length) el.childNodes[i].style.opacity = val
    }

  })


  function initVML() {

    /* Utility function to create a VML tag */
    function vml(tag, attr) {
      return createEl('<' + tag + ' xmlns="urn:schemas-microsoft.com:vml" class="spin-vml">', attr)
    }

    // No CSS transforms but VML support, add a CSS rule for VML elements:
    sheet.addRule('.spin-vml', 'behavior:url(#default#VML)')

    Spinner.prototype.lines = function(el, o) {
      var r = o.length+o.width
        , s = 2*r

      function grp() {
        return css(
          vml('group', {
            coordsize: s + ' ' + s,
            coordorigin: -r + ' ' + -r
          }),
          { width: s, height: s }
        )
      }

      var margin = -(o.width+o.length)*2 + 'px'
        , g = css(grp(), {position: 'absolute', top: margin, left: margin})
        , i

      function seg(i, dx, filter) {
        ins(g,
          ins(css(grp(), {rotation: 360 / o.lines * i + 'deg', left: ~~dx}),
            ins(css(vml('roundrect', {arcsize: o.corners}), {
                width: r,
                height: o.width,
                left: o.radius,
                top: -o.width>>1,
                filter: filter
              }),
              vml('fill', {color: getColor(o.color, i), opacity: o.opacity}),
              vml('stroke', {opacity: 0}) // transparent stroke to fix color bleeding upon opacity change
            )
          )
        )
      }

      if (o.shadow)
        for (i = 1; i <= o.lines; i++)
          seg(i, -2, 'progid:DXImageTransform.Microsoft.Blur(pixelradius=2,makeshadow=1,shadowopacity=.3)')

      for (i = 1; i <= o.lines; i++) seg(i)
      return ins(el, g)
    }

    Spinner.prototype.opacity = function(el, i, val, o) {
      var c = el.firstChild
      o = o.shadow && o.lines || 0
      if (c && i+o < c.childNodes.length) {
        c = c.childNodes[i+o]; c = c && c.firstChild; c = c && c.firstChild
        if (c) c.opacity = val
      }
    }
  }

  var probe = css(createEl('group'), {behavior: 'url(#default#VML)'})

  if (!vendor(probe, 'transform') && probe.adj) initVML()
  else useCssAnimations = vendor(probe, 'animation')

  return Spinner

}));
