'use strict';

angular.module('app').service('users', ['$http', function($http) {
	this.getCurrent = function() {
		return $http.get('/api/user');
	};
	this.get = function(host, login) {
		return $http.get('/api/users/'+host+'/'+login);
	};
	this.create = function(host, login) {
		return $http.post('/api/users/'+host+'/'+login);
	};
	this.delete = function(host, login) {
		return $http.delete('/api/users/'+host+'/'+login);
	};
}]);