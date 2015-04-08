'use strict';

(function () {

  /**
   * fromNow is a helper function that returns a human readable
   * string for the elapsed time between the given unix date and the
   * current time (ex. 10 minutes ago).
   */
	function fromNow() {
		return function(date) {
			if (!date) {
				return;
			}
 			return moment(new Date(date*1000)).fromNow();
		}
	}

	/**
	 * toDuration is a helper function that returns a human readable
	 * string for the given duration in seconds (ex. 1 hour and 20 minutes).
	*/
	function toDuration() {
		return function(seconds) {
			return moment.duration(seconds, "seconds").humanize();
		}
	}

 	/**
	 * toDate is a helper function that returns a human readable
	 * string gor the given unix date.
	*/
	function toDate() {
		return function(date) {
			return moment(new Date(date*1000)).format('ll');
		}
	}

	angular
		.module('drone')
		.filter('fromNow', fromNow)
		.filter('toDate', toDate)
		.filter('toDuration', toDuration)

})();
