'use strict';

angular.module('app').service('userService', function($q, $http, $timeout) {
	return{
		user : null,
		getCurrent : function() {
			var _this = this;
			var defer = $q.defer();

			// if the user is already authenticated
			if (_this.user != null) {
				defer.resolve(_this.user); 
			}

			// else we need to fetch from the server
			$http({method: 'GET', url: '/v1/user'}).
				then(function(data) {
						_this.user=data;
						defer.resolve(_this.user);
					}).
					error(function(data, status) {
						_this.user=null;
						defer.resolve();
					});

			// returns a promise that this will complete
			// at some future time.
			return defer.promise;
		}
	}
});