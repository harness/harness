'use strict';

angular.module('app').service('users', ['$http', function($http) {
	this.getCurrent = function() {
		return $http.get('/v1/user');
	};
	this.get = function(host, login) {
		return $http.get('/v1/users/'+host+'/'+login);
	};
	this.create = function(host, login) {
		return $http.post('/v1/users/'+host+'/'+login);
	};
	this.delete = function(host, login) {
		return $http.delete('/v1/users/'+host+'/'+login);
	};
}]);