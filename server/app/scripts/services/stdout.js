'use strict';

angular.module('app').service('stdout', ['$window', function($window) {
	var callback = undefined;
	var websocket = undefined;

	this.subscribe = function(path, _callback) {
		callback = _callback;

		var proto = ($window.location.protocol == 'https:' ? 'wss' : 'ws');
		var route = [proto, "://", $window.location.host, '/ws/stdout/', path].join('');

		websocket = new WebSocket(route);
		websocket.onmessage = function(event) {
			if (callback != undefined) {
				callback(event.data);
			}
		};
		websocket.onclose = function(event) {
			console.log('websocket closed at '+path);
		};
	};

	this.unsubscribe = function() {
		callback = undefined;
		if (websocket != undefined) {
			websocket.close();
		}
	};
}]);
