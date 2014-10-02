'use strict';

// Service facilitates interaction with the repository API.
angular.module('app').service('repos', ['$q', '$http', function($q, $http) {

	// Gets a repository by host, owner and name.
	// @deprecated
	this.getRepo = function(host, owner, name) {
			var defer = $q.defer();
			var route = '/api/repos/'+host+'/'+owner+'/'+name;
			$http.get(route).success(function(data){
				defer.resolve(data);
			});
			return defer.promise;
	};

	// Gets a repository by host, owner and name.
	this.get = function(host, owner, name) {
		return $http.get('/api/repos/'+host+'/'+owner+'/'+name);
	};

	// Gets a repository by host, owner and name.
	this.commits = function(host, owner, name) {
		return $http.get('/api/repos/'+host+'/'+owner+'/'+name+'/commits');
	};

	// Updates an existing repository
	this.update = function(repo) {
		// todo(bradrydzewski) add repo to the request body
		return $http.post('/api/repos/'+repo.host+'/'+repo.owner+'/'+repo.name);
	};

	// Activates a repository on the backend, registering post-commit
	// hooks with the remote hosting service (ie github).
	this.activate = function(repo) {
		// todo(bradrydzewski) add repo to the request body
		return $http.post('/api/repos/'+repo.host+'/'+repo.owner+'/'+repo.name);
	};

	// Deactivate a repository sets the active flag to false, instructing
	// the system to ignore all post-commit hooks for the repository.
	this.deactivate = function(repo) {
		return $http.delete('/api/repos/'+repo.host+'/'+repo.owner+'/'+repo.name);
	};
}]);