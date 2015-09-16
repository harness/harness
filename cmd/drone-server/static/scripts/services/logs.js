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
			return $http.get('api/repos/'+repoName+'/logs/'+number+'/'+step);
		};

		var callback,
			events,
			token = localStorage.getItem('access_token');

		this.subscribe = function (repoName, number, step, _callback) {
			callback = _callback;

			var route = ['api/stream/', repoName, '/', number, '/', step, '?access_token=', token].join('')
			events = new EventSource(route, { withCredentials: true });
			events.onmessage = function (event) {
				if (callback !== undefined) {
					callback(event.data);
				}
			};
			events.onerror = function (event) {
				callback = undefined;
				if (events !== undefined) {
					events.close();
					events = undefined;
				}
				console.log('user event stream closed due to error.', event);
			};
		};

		this.unsubscribe = function () {
			callback = undefined;
			if (events !== undefined) {
				events.close();
				events = undefined;
			}
		};
	}

	angular
		.module('drone')
		.service('logs', LogService);
})();
