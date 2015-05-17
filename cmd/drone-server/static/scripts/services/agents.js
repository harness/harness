'use strict';

(function () {

	/**
	 * The Agent provides access to build agent
	 * data and management using REST API calls.
	 */
	function AgentService($http) {

		/**
		 * Gets an agent token.
		 */
		this.getToken = function() {
			return $http.get('/api/agents/token');
		};
	}

	angular
		.module('drone')
		.service('agents', AgentService);
})();
