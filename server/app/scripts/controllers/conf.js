'use strict';

angular.module('app').controller("ConfigController", function($scope, $http, remotes) {
	$scope.state=0;

	$scope.user = remotes.get().success(function (data) {
			$scope.remotes = (typeof data==="string")?[]:data;
			$scope.state = 1;

			// loop through the remotes and add each
			// remote to the page scope.
			for (remote in $scope.remotes) {
				switch (remote.type) {
				case 'github.com':
					$scope.github = remote;
					break;

				case 'enterprise.github.com':
					$scope.githubEnterprise = remote;
					break;

				case 'gitlab.com':
					$scope.gitlab = remote;
					break;

				case 'bitbucket.org':
					$scope.bitbucket = remote;
					break;

				case 'stash.atlassian.com':
					$scope.stash = remote;
					break;
				case 'gogs':
					$scope.gogs = remote;
					break;
				}
			}
		})
		.error(function (error) {
			$scope.remotes = [];
			$scope.state = 1;
		});
});
