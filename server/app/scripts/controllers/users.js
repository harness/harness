'use strict';

angular.module('app').controller("UsersController", function($scope, $http, user) {

	$scope.user = user;

	$http({method: 'GET', url: '/api/users'}).
		success(function(data, status, headers, config) {
			$scope.users = data;
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});
});

angular.module('app').controller("UserAddController", function($scope, $http, users) {
	// set the default host to github ... however ...
	// eventually we can improve this logic to use the hostname
	// of the currently authenticated user.
	$scope.host='github.com';
	$scope.name='';

	$scope.create = function() {
		users.create($scope.host, $scope.name).success(function () {
				window.location.href="/admin/users";
			})
			.error(function (error) {
				console.log(error);
			});
	};
});

angular.module('app').controller("UserEditController", function($scope, $http, $routeParams, users) {

	var host = $routeParams.host;
	var name = $routeParams.login;

	users.get(host, name).success(function (user) {
			$scope.account = user;
			$scope.state = 1;
		})
		.error(function (error) {
			$scope.account = undefined;
			$scope.state = 1;
		});

	$scope.delete = function() {
		users.delete(host, name).success(function () {
				window.location.href="/admin/users";
			})
			.error(function (error) {
				console.log(error);
			});
	};
});