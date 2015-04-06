'use strict';

angular.module('app').controller("LoginController", function($scope, $http, remotes) {
	$scope.state=0
	$scope.user = remotes.getLogins().success(function (data) {
			$scope.remotes = (typeof data==="string")?[]:data;
			$scope.state = 1;
		})
		.error(function (error) {
			$scope.remotes = [];
			$scope.state = 1;
		});
});