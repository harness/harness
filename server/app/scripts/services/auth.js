'use strict;'

angular.module('app').service('authService', function($q, $http) {
	return{
		user : null,
		// getUser will retrieve the currently authenticated
		// user from the session. If no user is found a 401
		// Not Authorized status will be returned.
		getUser : function() {
			var _this = this;
			var defer = $q.defer();

			// if the user is already authenticated
			if (_this.user != null) {
				defer.resolve(_this.user); 
			}

			// else we need to fetch from the server
			$http({method: 'GET', url: '/api/user'}).
				success(function(data) {
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