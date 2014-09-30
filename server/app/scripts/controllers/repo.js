'use strict';

angular.module('app').controller("RepoController", function($scope, $http, $routeParams, $route, repos, feed, repo) {
	$scope.repo = repo;

	// subscribes to the global feed to receive
	// build status updates.
	feed.subscribe(function(item) {
		if (item.repo.host  == repo.host  &&
			item.repo.owner == repo.owner &&
			item.repo.name  == repo.name) {
			// display a toast message with the
			// commit details, allowing the user to
			// reload the page.
			$scope.msg = item;
			$scope.$apply();
		} else {
			// we trigger a toast (or html5) notification so the
			// user is aware another build started

		}
	});


	// load the repo commit feed
	repos.feed(repo.host, repo.owner, repo.name).success(function (feed) {
			$scope.commits = (typeof feed==='string')?[]:feed;
			$scope.state = 1;
		})
		.error(function (error) {
			$scope.commits = undefined;
			$scope.state = 1;
		});

	//$http({method: 'GET', url: '/v1/repos/'+repo.host+'/'+repo.owner+"/"+repo.name+"/feed"}).
	//	success(function(data, status, headers, config) {
	//		$scope.commits = (typeof data==='string')?[]:data;
	//	}).
	//	error(function(data, status, headers, config) {
	//		console.log(data);
	//	});

	$scope.activate = function() {
		// request to create a new repository
		$http({method: 'POST', url: '/api/repos/'+repo.host+'/'+repo.owner+"/"+repo.name }).
			success(function(data, status, headers, config) {
				$scope.repo = data;
			}).
			error(function(data, status, headers, config) {
				$scope.failure = data;
			});
	};


	$scope.reload = function() {
		$route.reload();
	};

	//$scope.activate = function() {
	//	repos.activate($scope.host, $scope.name).success(function () {
	//			window.location.href="/admin/users";
	//		})
	//		.error(function (error) {
	//			console.log(error);
	//		});
	//};

});




angular.module('app').controller("RepoConfigController", function($scope, $http, $routeParams, user) {
	$scope.user = user;

	var remote = $routeParams.remote;
	var owner  = $routeParams.owner;
	var name   = $routeParams.name;

	// load the repo meta-data
	// request admin details for the repository as well.
	$http({method: 'GET', url: '/api/repos/'+remote+'/'+owner+"/"+name+"?admin=1"}).
		success(function(data, status, headers, config) {
			$scope.repo = data;
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});

	$scope.save = function() {
		// request to create a new repository
		$http({method: 'PUT', url: '/api/repos/'+remote+'/'+owner+"/"+name, data: $scope.repo }).
			success(function(data, status, headers, config) {
				delete $scope.failure;
			}).
			error(function(data, status, headers, config) {
				$scope.failure = data;
			});
	};
});