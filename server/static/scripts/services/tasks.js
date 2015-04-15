'use strict';

(function () {

	/**
	 * The TaskService provides access to build
	 * task data using REST API calls.
	 */
	function TaskService($http, $window) {

		/**
		 * Gets a list of builds.
		 *
		 * @param {string} Name of the repository.
		 * @param {number} Number of the build.
		 */
		this.list = function(repoName, number) {
			return $http.get('/api/repos/'+repoName+'/builds/'+number+'/tasks');
		};

		/**
		 * Gets a task.
		 *
		 * @param {string} Name of the repository.
		 * @param {number} Number of the build.
		 * @param {number} Number of the task.
		 */
		this.get = function(repoName, number, step) {
			return $http.get('/api/repos/'+repoName+'/builds/'+number+'/tasks/'+step);
		};

		/**
		 * Gets a task.
		 *
		 * @param {string} Name of the repository.
		 * @param {number} Number of the build.
		 * @param {number} Number of the task.
		 */
		this.get = function(repoName, number, step) {
			return $http.get('/api/repos/'+repoName+'/builds/'+number+'/tasks/'+step);
		};
	}

	angular
		.module('drone')
		.service('tasks', TaskService);
})();
