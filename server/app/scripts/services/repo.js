'use strict';

angular.module('app').service('repoService', function($q, $http) {
	return{
		getRepo : function(host, owner, name) {
			var defer = $q.defer();
			var route = '/v1/repos/'+host+'/'+owner+'/'+name;
			$http.get(route).success(function(data){
				defer.resolve(data);
			});
			return defer.promise;
		}
	}
});
