'use strict';

angular.module('app').service('feed', ['$http', '$window', function($http, $window) {

	var proto = ($window.location.protocol == 'https:' ? 'wss' : 'ws');
	var route = [proto, "://", $window.location.host, '/ws/user'].join('');

	var wsCallback = undefined;
	var ws = new WebSocket(route);
	ws.onmessage = function(event) {
		var data = angular.fromJson(event.data);
		if (wsCallback != undefined) {
			wsCallback(data);
		}
	};

	this.subscribe = function(callback) {
		wsCallback = callback;
	};

	this.unsubscribe = function() {
		ws.close();
	};
}]);
