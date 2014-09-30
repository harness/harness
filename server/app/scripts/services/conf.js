'use strict';

angular.module('app').service('confService', function($q, $http) {
	return{
		getConfig : function() {
			var defer = $q.defer();
			var route = '/api/config';
			$http.get(route).success(function(data){
				defer.resolve(data);
			});
			return defer.promise;
		}
	}
});