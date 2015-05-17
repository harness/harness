'use strict';

(function () {

	/**
	 * The BuildsService provides access to build
	 * data using REST API calls.
	 */
	function BuildService($http, $window) {

		/**
		 * Gets a list of builds.
		 *
		 * @param {string} Name of the repository.
		 */
		this.list = function(repoName) {
			return $http.get('/api/repos/'+repoName+'/builds');
		};

		/**
		 * Gets a build.
		 *
		 * @param {string} Name of the repository.
		 * @param {number} Number of the build.
		 */
		this.get = function(repoName, buildNumber) {
			return $http.get('/api/repos/'+repoName+'/builds/'+buildNumber);
		};

		/**
		 * Restarts a build.
		 *
		 * @param {string} Name of the repository.
		 * @param {number} Number of the build.
		 */
		this.restart = function(repoName, buildNumber) {
			return $http.post('/api/repos/' + repoName+'/builds/'+buildNumber);
		};

		/**
		 * Cancels a running build.
		 *
		 * @param {string} Name of the repository.
		 * @param {number} Number of the build.
		 */
		this.cancel = function(repoName, buildNumber) {
			return $http.delete('/api/repos/'+repoName+'/builds/'+buildNumber);
		};
	}

	angular
		.module('drone')
		.service('builds', BuildService);
})();
