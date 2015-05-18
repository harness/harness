'use strict';

(function () {

	/**
	 * The Agent provides access to build agent
	 * data and management using REST API calls.
	 */
	function AgentService($http) {

		/**
		 * Gets an agent list.
		 */
		this.getAgents = function() {
			return $http.get('/api/agents');
		};

		this.deleteAgent = function(agent) {
			return $http.delete('/api/agents/'+agent.id);
		};

		this.postAgent = function(agent) {
			return $http.post('/api/agents', agent);
		};

	}

	angular
		.module('drone')
		.service('agents', AgentService);
})();
