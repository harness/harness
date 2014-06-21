'use strict';

angular.module('app').controller("HomeController", function($scope, $http, user) {

	$scope.user = user;

	$http({method: 'GET', url: '/v1/user/feed'}).
		success(function(data, status, headers, config) {
			$scope.feed = (typeof data==='string')?[]:data;
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});
});
