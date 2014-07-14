'use strict';

angular.module('app').controller("HomeController", function($scope, $http, feed) {

	feed.subscribe(function(item) {
		// todo toast notification
	});

	$http({method: 'GET', url: '/v1/user/feed'}).
		success(function(data, status, headers, config) {
			$scope.feed = (typeof data==='string')?[]:data;
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});

	$http({method: 'GET', url: '/v1/user/repos'}).
		success(function(data, status, headers, config) {
			$scope.repos = (typeof data==='string')?[]:data;
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});
});
