'use strict';

// Service facilitates interaction with the remote API.
angular.module('app').service('remotes', ['$http', function($http) {

	this.get = function() {
		return $http.get('/v1/remotes');
	};

	this.getLogins = function() {
		return $http.get('/v1/logins');
	};
}]);