(function () {

	/**
	 * BuildsCtrl responsible for rendering the repo's
	 * recent build history.
	 */
	function BuildsCtrl($scope, $routeParams, builds, repos, users, feed) {

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

		feed.subscribe(function(event) {
			if (event.repo.full_name !== fullName) {
				return; // ignore
			}
			// update repository
			$scope.repo = event.repo;
			$scope.apply();
			
			if (event.build.number !== parseInt(number)) {
				return; // ignore
			}
			// update the build status
			$scope.build.state = event.build.state;
			$scope.build.started = event.build.started;
			$scope.build.finished = event.build.finished;
			$scope.build.duration = event.build.duration;
			$scope.$apply();
		});
	}

	/**
	 * BuildCtrl responsible for rendering a build.
	 */
	function BuildCtrl($scope, $routeParams, logs, builds, repos, users, feed) {

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

		$scope.restart = function() {
			builds.restart(fullName, number).catch(function(err){
				$scope.error = err;
			});
		};

		$scope.cancel = function() {
			builds.cancel(fullName, number).catch(function(err) {
				$scope.error = err;
			});
		};

		feed.subscribe(function(event) {
			if (event.repo.full_name !== fullName) {
				return; // ignore
			}
			if (event.build.number !== parseInt(number)) {
				return; // ignore
			}
			// update the build status
			$scope.build.state = event.build.state;
			$scope.build.started = event.build.started;
			$scope.build.finished = event.build.finished;
			$scope.build.duration = event.build.duration;
			$scope.$apply();

			if (!event.task || event.task.number !== step) {
				return; // ignore
			}
			// update the task status
			$scope.task.state = event.task.state;
			$scope.task.started = event.task.started;
			$scope.task.finished = event.task.finished;
			$scope.task.duration = event.task.duration;
			$scope.task.exit_code = event.task.exit_code;
			$scope.$apply();
		});

		// var convert = new Filter({stream:true,newline:false});
		// var term = document.getElementById("term")
		// var stdout = document.getElementById("stdout").innerText.split("\n")
		// stdout.forEach(function(line, i) {
		// 	setTimeout(function () {
		// 		term.innerHTML += convert.toHtml(line+"\n");
		// 	}, i*i);
		// });
	}

	angular
		.module('drone')
		.controller('BuildCtrl', BuildCtrl)
		.controller('BuildsCtrl', BuildsCtrl);
})();
