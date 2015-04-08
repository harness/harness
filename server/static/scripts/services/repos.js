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
			return $http.put('/api/repos/'+repo.full_name, repo);
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
			return $http.post('/api/subscribers/'+repoName);
		};

		/**
		 * Unwatch a repository.
		 *
		 * @param {string} Name of the repository.
		 */
		this.unwatch = function(repoName) {
			return $http.delete('/api/subscribers/'+repoName);
		};
	}

	angular
		.module('drone')
		.service('repos', RepoService);
})();