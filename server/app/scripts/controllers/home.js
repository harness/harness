'use strict';

angular.module('app').controller("HomeController", function($scope, $http, user, websocket) {

	$scope.user = user;

	websocket.subscribeRepos(function(repos) {
		console.log(repos);
	});

	$http({method: 'GET', url: '/v1/user/feed'}).
		success(function(data, status, headers, config) {
			$scope.feed = (typeof data==='string')?[]:data;
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});
});
