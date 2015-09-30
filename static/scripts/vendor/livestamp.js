// Livestamp.js / v2.0.0 / (c) 2015 Matt Bradley / MIT License
(function (plugin) {
  if (typeof define === 'function' && define.amd) {
    // AMD. Register as an anonymous module.
    define(['jquery', 'moment'], plugin);
  } else {
    // Browser globals
    plugin(jQuery, moment);
  }
}(function($, moment) {
  var updateInterval = 1e3,
      paused = false,
      $livestamps = $([]),

  init = function() {
    livestampGlobal.resume();
  },

  prep = function($el, timestamp) {
    var oldData = $el.data('livestampdata');
    if (typeof timestamp == 'number')
      timestamp *= 1e3;

    $el.removeAttr('data-livestamp')
      .removeData('livestamp');

    timestamp = moment(timestamp);
    if (moment.isMoment(timestamp) && !isNaN(+timestamp)) {
      var newData = $.extend({ }, { 'original': $el.contents() }, oldData);
      newData.moment = moment(timestamp);

      $el.data('livestampdata', newData).empty();
      $livestamps.push($el[0]);
    }
  },

  run = function() {
    if (paused) return;
    livestampGlobal.update();
    setTimeout(run, updateInterval);
  },

  livestampGlobal = {
    update: function() {
      $('[data-livestamp]').each(function() {
        var $this = $(this);
        prep($this, $this.data('livestamp'));
      });

      var toRemove = [];
      $livestamps.each(function() {
        var $this = $(this),
            data = $this.data('livestampdata');

        if (data === undefined)
          toRemove.push(this);
        else if (moment.isMoment(data.moment)) {
          var from = $this.html(),
              to = data.moment.fromNow();

          if (from != to) {
            var e = $.Event('change.livestamp');
            $this.trigger(e, [from, to]);
            if (!e.isDefaultPrevented())
              $this.html(to);
          }
        }
      });

      $livestamps = $livestamps.not(toRemove);
      delete $livestamps.prevObject
    },

    pause: function() {
      paused = true;
    },

    resume: function() {
      paused = false;
      run();
    },

    interval: function(interval) {
      if (interval === undefined)
        return updateInterval;
      updateInterval = interval;
    }
  },

  livestampLocal = {
    add: function($el, timestamp) {
      if (typeof timestamp == 'number')
        timestamp *= 1e3;
      timestamp = moment(timestamp);

      if (moment.isMoment(timestamp) && !isNaN(+timestamp)) {
        $el.each(function() {
          prep($(this), timestamp);
        });
        livestampGlobal.update();
      }

      return $el;
    },

    destroy: function($el) {
      $livestamps = $livestamps.not($el);
      $el.each(function() {
        var $this = $(this),
            data = $this.data('livestampdata');

        if (data === undefined)
          return $el;

        $this
          .html(data.original ? data.original : '')
          .removeData('livestampdata');
      });

      return $el;
    },

    isLivestamp: function($el) {
      return $el.data('livestampdata') !== undefined;
    }
  };

  $.livestamp = livestampGlobal;
  $(init);
  $.fn.livestamp = function(method, options) {
    if (!livestampLocal[method]) {
      options = method;
      method = 'add';
    }

    return livestampLocal[method](this, options);
  };
}));