'use strict';

angular.module('app').controller("LoginController", function($scope, $http, remotes) {
	$scope.state=0
	$scope.user = remotes.getLogins().success(function (data) {
			$scope.remotes = (typeof data==="string")?[]:data;
			$scope.oauth_remotes = $scope.remotes.filter(function(r) { return r.type != 'gitlab.com' })
			$scope.manual_remotes = $scope.remotes.filter(function(r) { return r.type == 'gitlab.com' })
			$scope.state = 1;
		})
		.error(function (error) {
			$scope.remotes = [];
			$scope.oauth_remotes = [];
			$scope.manual_remotes = [];
			$scope.state = 1;
		});
});