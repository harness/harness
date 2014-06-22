'use strict';

angular.module('app').service('websocket', function($q, $http, $window) {
	var wsCallback = undefined;
	var ws = new WebSocket('ws://localhost:8080/ws/user');
	ws.onmessage = function(event) {
		var data = angular.fromJson(event.data);
		if (wsCallback != undefined) {
			wsCallback(data);
		}
	};
	return {
		subscribeRepos: function(callback) {
			wsCallback = callback;
		}
	};
});

