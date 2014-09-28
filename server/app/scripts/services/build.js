'use strict';

// Service facilitates interaction with the repository API.
angular.module('app').service('builds', ['$q', '$http', function($q, $http) {
	// Gets a repository by host, owner and name.
	this.feed = function(host, owner, name, branch, sha) {
		return $http.get('/v1/repos/'+host+'/'+owner+'/'+name+'/branches/'+branch+'/commits/'+sha+'/builds');
	};

	this.get = function(host, owner, name, branch, sha, index) {
		return $http.get('/v1/repos/'+host+'/'+owner+'/'+name+'/branches/'+branch+'/commits/'+sha+'/builds/'+index);
	};

}]);