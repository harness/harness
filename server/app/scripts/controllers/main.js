'use strict';

angular.module('app').controller("MainCtrl", function($scope, $http, users) {
	$scope.state=0
	$scope.user = users.getCurrent().success(function (user) {
			$scope.user = user;
			$scope.state = 1;
		})
		.error(function (error) {
			$scope.user = undefined;
			$scope.state = 1;
		});
});
