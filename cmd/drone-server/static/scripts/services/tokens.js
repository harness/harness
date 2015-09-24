'use strict';

(function () {

	/**
	 * The TokenService provides access to user token
	 * data using REST API calls.
	 */
	function TokenService($http, $window) {

		/**
		 * Generates a user API token.
		 */
		this.post = function(token) {
			return $http.post('api/user/token');
		};
	}

	angular
		.module('drone')
		.service('tokens', TokenService);
})();