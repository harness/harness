'use strict';

angular.module('app').controller("RepoController", function($scope, $http, $routeParams, $route, repos, feed, repo) {
	$scope.repo = repo;
	$scope.activating = false;

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
	repos.commits(repo.host, repo.owner, repo.name).success(function (commits) {
			$scope.commits = (typeof commits==='string')?[]:commits;
			$scope.state = 1;
		})
		.error(function (error) {
			$scope.commits = undefined;
			$scope.state = 1;
		});

	//$http({method: 'GET', url: '/api/repos/'+repo.host+'/'+repo.owner+"/"+repo.name+"/feed"}).
	//	success(function(data, status, headers, config) {
	//		$scope.commits = (typeof data==='string')?[]:data;
	//	}).
	//	error(function(data, status, headers, config) {
	//		console.log(data);
	//	});

	$scope.activate = function() {
		$scope.activating = true;

		// request to create a new repository
		$http({method: 'POST', url: '/api/repos/'+repo.host+'/'+repo.owner+"/"+repo.name }).
			success(function(data, status, headers, config) {
				$scope.repo = data;
				$scope.activating = false;
			}).
			error(function(data, status, headers, config) {
				$scope.failure = data;
				$scope.activating = false;
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




angular.module('app').controller("RepoConfigController", function($scope, $http, $timeout, $routeParams, user) {
	$scope.user = user;
	$scope.saving = false;

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
		$scope.saving = true;

		// request to create a new repository
		$http({method: 'PUT', url: '/api/repos/'+remote+'/'+owner+"/"+name, data: $scope.repo }).
			success(function(data, status, headers, config) {
				delete $scope.failure;

				// yes, for UX reasons we make this request look like it
				// is taking longer than it really is. Otherwise the loading
				// button just instantly flickers.
				$timeout(function(){ 
						$scope.saving = false;
					}, 1500);
			}).
			error(function(data, status, headers, config) {
				$scope.failure = data;
				$scope.saving = false;
			});
		
	};
});