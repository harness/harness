'use strict';

angular.module('app').controller("UsersController", function($scope, $http, user) {

	$scope.user = user;

	$http({method: 'GET', url: '/v1/users'}).
		success(function(data, status, headers, config) {
			$scope.users = data;
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});
});