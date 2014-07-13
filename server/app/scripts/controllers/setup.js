'use strict';

angular.module('app').controller("SetupController", function($scope, $http, $routeParams, $window) {

	// create a remote that will be populated
	// and persisted to the database.
	$scope.remote = {};
	$scope.remote.type = $routeParams.remote;
	$scope.remote.register = false;
	$scope.window = $window

	// pre-populate the form if the remote
	// type is selected and is a cloud service
	// with a known URL and standard configuration.
	switch($scope.remote.type) {
	case 'github.com':
		$scope.remote.type = "github.com"
		$scope.remote.url = "https://github.com";
		$scope.remote.api = "https://api.github.com";
		break;
	case 'bitbucket.org':
		$scope.remote.url = "https://bitbucket.org";
		$scope.remote.api = "https://bitbucket.org";
		break;
	}

	$scope.save = function() {
		// request to create a new repository
		$http({method: 'POST', url: '/v1/remotes', data: $scope.remote }).
			success(function(data, status, headers, config) {
				delete $scope.failure;
				$scope.remote = data;
				$window.location.href="/login/"+data.type;
			}).
			error(function(data, status, headers, config) {
				$scope.failure = data;
				console.log('failure', $scope.failure);
			});
	};
});