'use strict';

angular.module('app').service('notify', ['$window', '$timeout', function($window, $timeout) {

	this.supported = function() {
		return ("Notification" in $window)
	}

	this.granted = function() {
		return ("Notification" in $window) && Notification.permission === "granted";
	}

	this.requestPermission = function() {
		Notification.requestPermission();
	}

	this.send = function(title, opts) {
		if ("Notification" in $window) {
			var n = new Notification(title, opts);
			$timeout(function() { n.close(); }, 10000);
		}
	};
}]);
