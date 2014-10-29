'use strict';

angular.module('app').controller("HomeController", function($scope, $http, $location, feed) {

	feed.subscribe(function(item) {
		// todo toast notification
	});

	$http({method: 'GET', url: '/api/user/feed'}).
		success(function(data, status, headers, config) {
			$scope.feed = (typeof data==='string')?[]:data;
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});

	$scope.syncUser = function() {
		$http({method: 'POST', url: '/api/user/sync' }).success(function(data){
			$location.search('return_to', $location.$$path).path('/sync')
		}).error(function(data, status){
			if (status == 409) {
				$scope.msg = 'already'
			} else {
				$scope.msg = 'bad'
			}
			$scope.$apply();
		});
	}

	$http({method: 'GET', url: '/api/user/repos'}).
		success(function(data, status, headers, config) {
			$scope.repos = (typeof data==='string')?[]:data;
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});
});
