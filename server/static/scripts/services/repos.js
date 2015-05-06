'use strict';

(function () {

	/**
	 * The RepoService provides access to repository
	 * data using REST API calls.
	 */
	function RepoService($http, $window) {

		var callback,
			websocket,
			token = localStorage.getItem('access_token');

		/**
		 * Gets a list of all repositories.
		 */
		this.list = function() {
			return $http.get('/api/user/repos');
		};

		/**
		 * Gets a repository by name.
		 *
		 * @param {string} Name of the repository.
		 */
		this.get = function(repoName) {
			return $http.get('/api/repos/'+repoName);
		};

		/**
		 * Creates a new repository.
		 *
		 * @param {object} JSON representation of a repository.
		 */
		this.post = function(repoName) {
			return $http.post('/api/repos/' + repoName);
		};

		/**
		 * Updates an existing repository.
		 *
		 * @param {object} JSON representation of a repository.
		 */
		this.update = function(repo) {
			return $http.patch('/api/repos/'+repo.full_name, repo);
		};

		/**
		 * Deletes a repository.
		 *
		 * @param {string} Name of the repository.
		 */
		this.delete = function(repoName) {
			return $http.delete('/api/repos/'+repoName);
		};

		/**
		 * Watch a repository.
		 *
		 * @param {string} Name of the repository.
		 */
		this.watch = function(repoName) {
			return $http.post('/api/repos/'+repoName+'/watch');
		};

		/**
		 * Unwatch a repository.
		 *
		 * @param {string} Name of the repository.
		 */
		this.unwatch = function(repoName) {
			return $http.delete('/api/repos/'+repoName+'/unwatch');
		};


		var callback,
			websocket,
			token = localStorage.getItem('access_token');

		/**
		 * Subscribes to a live update feed for a repository
		 *
		 * @param {string} Name of the repository.
		 */
		this.subscribe = function(repo, _callback) {
			callback = _callback;

			var proto = ($window.location.protocol === 'https:' ? 'wss' : 'ws'),
				route = [proto, "://", $window.location.host, '/api/stream/'+ repo +'?access_token=', token].join('');

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
		.service('repos', RepoService);
})();
