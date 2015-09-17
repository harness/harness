'use strict';

(function () {

	function FeedService($http, $window, $state) {

		var callback,
			websocket,
			token = localStorage.getItem('access_token');

		this.subscribe = function(_callback) {
			callback = _callback;

			var proto = ($window.location.protocol === 'https:' ? 'wss' : 'ws'),
				route = [proto, "://", $window.location.host, $state.href('app.index'), 'api/stream/user?access_token=', token].join('');

			websocket = new WebSocket(route);
			websocket.onmessage = function (event) {
				if (callback !== undefined) {
					callback(angular.fromJson(event.data));
				}
			};
			websocket.onclose = function (event) {
				console.log('user websocket closed');
			};
		};

		this.unsubscribe = function() {
			callback = undefined;
			if (websocket !== undefined) {
				websocket.close();
				websocket = undefined;
			}
		};
	}

	angular
		.module('drone')
		.service('feed', FeedService);
})();
