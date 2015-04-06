'use strict';

angular.module('app').controller("RepoController", function($scope, $filter, $http, $routeParams, $route, repos, feed, repo) {
	$scope.repo = repo;
	$scope.activating = false;
	$scope.build_filter = 'build_history';
	$scope.layout = 'grid';

	// subscribes to the global feed to receive
	// build status updates.
	feed.subscribe(function(item) {
		if (item.repo.host  == repo.host  &&
			item.repo.owner == repo.owner &&
			item.repo.name  == repo.name) {
			// display a toast message with the
			// commit details, allowing the user to
			// reload the page.

			// Try find an existing commit for this SHA. If found, replace it
			var sha_updated = $scope.commits.some(function(element, index) {
				if (element.sha == item.commit.sha)
					$scope.commits[index] = item.commit;
				return element.sha == item.commit.sha;
			});

			// Add a build message if the SHA couldn't be found and the new build status is 'Started'
			if ( ! sha_updated && item.commit.status == 'Started') {
				// $scope.commits.unshift(item.commit);
				$scope.msg = item;
			}

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

	$scope.setCommitFilter = function(filter) {
		$scope.build_filter = filter;
	}

	$scope.setLayout = function(layout) {
		$scope.layout = layout;
	}

	$scope.filteredCommits = function() {
		var filteredCommits;
		switch ($scope.build_filter) {
			// Latest commit for each branch (excluding PR branches)
			case 'branch_summary':
				filteredCommits = $filter('filter')($scope.commits, { pull_request: '' }, true);
				filteredCommits = $filter('unique')($scope.commits, 'branch');
				break;
			// Latest commit for each PR
			case 'pull_requests':
				filteredCommits = $filter('pullRequests')($scope.commits);
				filteredCommits = $filter('unique')(filteredCommits, 'pull_request');
				break;
			// All commits for a full build history
			default:
				filteredCommits = $scope.commits;
		}

		return filteredCommits;
	}
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