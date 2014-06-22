'use strict';

angular.module('app').controller("UserController", function($scope, $http, user, notify) {

	$scope.user = user;

	// get the user details
	$http({method: 'GET', url: '/v1/user'}).
		success(function(data, status, headers, config) {
			$scope.user = data;
			$scope.userTemp = {
				email : $scope.user.email,
				name  : $scope.user.name
			};
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});

	$scope.notifications = {}
	$scope.notifications.supported = notify.supported();
	$scope.notifications.granted = notify.granted();

	$scope.save = function() {
		// request to create a new repository
		$http({method: 'PUT', url: '/v1/user', data: $scope.userTemp }).
			success(function(data, status, headers, config) {
				delete $scope.failure;
				$scope.user = data;
			}).
			error(function(data, status, headers, config) {
				$scope.failure = data;
			});
	};
	$scope.cancel = function() {
		delete $scope.failure;
		$scope.userTemp = {
			email : $scope.user.email,
			name  : $scope.user.name
		};
	};
	$scope.enableNotifications = function() {
		notify.requestPermission();
	};
});