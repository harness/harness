(function () {

	/**
	 * CommitsCtrl responsible for rendering the repo's
	 * recent commit history.
	 */
	function CommitsCtrl($scope, $routeParams, builds, repos, users, logs) {

		var owner = $routeParams.owner;
		var name  = $routeParams.name;
		var fullName = owner+'/'+name;

		// Gets the currently authenticated user
		users.getCached().then(function(payload){
			$scope.user = payload.data;
		});

		// Gets a repository
		repos.get(fullName).then(function(payload){
			$scope.repo = payload.data;
		}).catch(function(err){
			$scope.error = err;
		});

		// Gets a list of commits
		builds.list(fullName).then(function(payload){
			$scope.builds = angular.isArray(payload.data) ? payload.data : [];
		}).catch(function(err){
			$scope.error = err;
		});

		$scope.watch = function(repo) {
			repos.watch(repo.full_name).then(function(payload) {
				$scope.repo.starred = true;
			});
		}

		$scope.unwatch = function(repo) {
			repos.unwatch(repo.full_name).then(function() {
				$scope.repo.starred = false;
			});
		}

		repos.subscribe(fullName, function(event) {
			var added = false;
			for (var i=0;i<$scope.builds.length;i++) {
				var build = $scope.builds[i];
				if (event.number !== build.number) {
					continue; // ignore
				}
				// update the build status
				$scope.builds[i] = event;
				$scope.$apply();
				added = true;
			}

			if (!added) {
				$scope.builds.push(event);
				$scope.$apply();
			}
		});
	}

	angular
		.module('drone')
		.controller('CommitsCtrl', CommitsCtrl)
})();
