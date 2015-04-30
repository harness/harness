'use strict';

(function () {

	/**
	 * The LogService provides access to build
	 * log data using REST API calls.
	 */
	function LogService($http, $window) {

		/**
		 * Gets a task logs.
		 *
		 * @param {string} Name of the repository.
		 * @param {number} Number of the build.
		 * @param {number} Number of the task.
		 */
		this.get = function(repoName, number, step) {
			return $http.get('/api/repos/'+repoName+'/logs/'+number+'/'+step);
		};

		var callback,
			websocket,
			token = localStorage.getItem('access_token');

		this.subscribe = function (repoName, number, step, _callback) {
			callback = _callback;

			var proto = ($window.location.protocol === 'https:' ? 'wss' : 'ws'),
				route = [proto, "://", $window.location.host, '/api/stream/logs/', repoName, '/', number, '/', step, '?access_token=', token].join('');

			websocket = new WebSocket(route);
			websocket.onmessage = function (event) {
				if (callback !== undefined) {
					callback(event.data);
				}
			};
			websocket.onclose = function (event) {
				console.log('logs websocket closed');
			};
		};

		this.unsubscribe = function () {
			callback = undefined;
			if (websocket !== undefined) {
				websocket.close();
				websocket = undefined;
			}
		};
	}

	angular
		.module('drone')
		.service('logs', LogService);
})();
