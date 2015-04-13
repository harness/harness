'use strict';

(function () {

	/**
	 * The TokenService provides access to user token
	 * data using REST API calls.
	 */
	function TokenService($http, $window) {

		/**
		 * Gets a list of all repositories.
		 */
		this.list = function() {
			return $http.get('/api/user/tokens');
		};

		/**
		 * Creates a new token.
		 *
		 * @param {object} JSON representation of a repository.
		 */
		this.post = function(token) {
			return $http.post('/api/user/tokens', token);
		};

		/**
		 * Deletes a repository.
		 *
		 * @param {string} Name of the repository.
		 */
		this.delete = function(token) {
			return $http.delete('/api/user/tokens/' + token.label);
		};
	}

	angular
		.module('drone')
		.service('tokens', TokenService);
})();