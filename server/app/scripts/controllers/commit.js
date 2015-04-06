/*global angular, Drone, console */
angular.module('app').controller("CommitController", function ($scope, $http, $route, $routeParams, stdout, feed) {
	'use strict';

	var remote = $routeParams.remote,
		owner  = $routeParams.owner,
		name   = $routeParams.name,
		branch = $routeParams.branch,
		commit = $routeParams.commit,
		// Create lineFormatter and outputElement since we need them anyway.
		lineFormatter = new Drone.LineFormatter(),
		outputElement = angular.element(document.querySelector('#output'));
		
	var connectRemoteConsole = function (id) {
		// Clear console output if connecting to new remote console (rebuild)
		if (!outputElement.html() !== 0) {
			outputElement.empty();
		}
		// Subscribe to stdout of the remote build
		stdout.subscribe(id, function (out) {
			// Append new output to console
			outputElement.append(lineFormatter.format(out));
			// Scroll if following
			if ($scope.following) {
				window.scrollTo(0, document.body.scrollHeight);
			}
		});
	};

	// Subscribe to feed so we can update gui if changes to the commit happen. (Build finished, Rebuild triggered, change from Pending to Started)
	feed.subscribe(function (item) {
		// If event is part of the active commit currently showing.
		if (item.commit.sha === commit &&
				item.commit.branch === branch) {
			// If new status is Started, connect to remote console to get live output
			if (item.commit.status === "Started") {
				connectRemoteConsole(item.commit.id);
			}
			$scope.commit = item.commit;
			$scope.$apply();

		} else {
			// we trigger an toast notification so the
			// user is aware another build started
			
		}
	});

	// Load the repo meta-data
	$http({method: 'GET', url: '/api/repos/' + remote + '/' + owner + "/" + name}).
		success(function (data, status, headers, config) {
			$scope.repo = data;
		}).
		error(function (data, status, headers, config) {
			console.log(data);
		});

	// Load the repo commit data
	$http({method: 'GET', url: '/api/repos/' + remote + '/' + owner + "/" + name + "/branches/" + branch + "/commits/" + commit}).
		success(function (data, status, headers, config) {
			$scope.commit = data;

			// If build has already finished, load console output from database
			if (data.status !== 'Started' && data.status !== 'Pending') {
				$http({method: 'GET', url: '/api/repos/' + remote + '/' + owner + "/" + name + "/branches/" + branch + "/commits/" + commit + "/console"}).
					success(function (data, status, headers, config) {
						outputElement.append(lineFormatter.format(data));
					}).
					error(function (data, status, headers, config) {
						console.log(data);
					});
				return;
			// If build is currently running, connect to remote console;
			} else if (data.status === 'Started') {
				connectRemoteConsole(data.id);
			}
		
		}).
		error(function (data, status, headers, config) {
			console.log(data);
		});

	$scope.following = false;
	$scope.follow = function () {
		$scope.following = true;
		window.scrollTo(0, document.body.scrollHeight);
	};
	$scope.unfollow = function () {
		$scope.following = false;
	};

	$scope.rebuildCommit = function () {
        $http({method: 'POST', url: '/api/repos/' + remote + '/' + owner + '/' + name + '/branches/' + branch + '/commits/' + commit + '?action=rebuild' });
	};
});
