'use strict';

angular.module('app').service('users', ['$http', function($http) {
	this.getCurrent = function() {
		return $http.get('/v1/user');
	};
}]);