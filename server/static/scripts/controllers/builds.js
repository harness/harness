(function () {

	/**
	 * BuildsCtrl responsible for rendering the repo's
	 * recent build history.
	 */
	function BuildsCtrl($scope, $routeParams, builds, repos, users) {

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

		// Gets a list of builds
		builds.list(fullName).then(function(payload){
			$scope.builds = angular.isArray(payload.data) ? payload.data : [];
		}).catch(function(err){
			$scope.error = err;
		});

		$scope.watch = function(repo) {
			repos.watch(repo.full_name).then(function(payload) {
				$scope.repo.subscription = payload.data;
			});
		}

		$scope.unwatch = function(repo) {
			repos.unwatch(repo.full_name).then(function() {
				delete $scope.repo.subscription;
			});
		}
	}

	/**
	 * BuildCtrl responsible for rendering a build.
	 */
	function BuildCtrl($scope, $routeParams, logs, builds, repos, users) {

		var step = parseInt($routeParams.step) || 1;
		var number = $routeParams.number;
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

		// Gets the build
		builds.get(fullName, number).then(function(payload){
			$scope.build = payload.data;
			$scope.task = payload.data.tasks[step-1];

			if ($scope.task.state === 'pending') {
				// do nothing
			} else if ($scope.task.state === 'running') {
				// stream the build
			} else {

				// fetch the logs for the finished build.
				logs.get(fullName, number, step).then(function(payload){
					$scope.logs = payload.data;
				}).catch(function(err){
					$scope.error = err;
				});
			}
		}).catch(function(err){
			$scope.error = err;
		});

	}

	angular
		.module('drone')
		.controller('BuildCtrl', BuildCtrl)
		.controller('BuildsCtrl', BuildsCtrl);
})();
