'use strict';

app.controller("BuildController", function($scope, $http, $routeParams, builds, feed, stdout) {
	var remote = $routeParams.remote;
	var owner  = $routeParams.owner;
	var name   = $routeParams.name;
	var branch = $routeParams.branch;
	var commit = $routeParams.commit;
	var build  = $routeParams.build;

	var lineFormatter = new Drone.LineFormatter();
	var el = document.querySelector('#output');

	$http({method: 'GET', url: '/v1/repos/'+remote+'/'+owner+"/"+name}).
		success(function(data, status, headers, config) {
			$scope.repo = data;
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});

	$http({method: 'GET', url: '/v1/repos/'+remote+'/'+owner+"/"+name+"/branches/"+branch+"/commits/"+commit}).
		success(function(data, status, headers, config) {
			$scope.commit = data;
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});

	builds.get(remote, owner, name, branch, commit, build).
		success(function (data) {
			$scope.build = data

			// Subscribe global updates
			feed.subscribe(function(item) {
				var feed_build = item.builds.filter(function(b) { return b.id == data.id; })[0]

				if (item.commit.sha    == commit &&
					item.commit.branch == branch &&
					feed_build != undefined) {
					$scope.build = feed_build;
					$scope.$apply();
				} else {
					// we trigger an toast notification so the
					// user is aware another build started
				}
			});

			// Fetch output
			if (data.status!='Started' && data.status!='Pending') {
				angular.element(el).append(lineFormatter.format(data.output));
			}

			// Subscribe build channel
			var path = data.commit_id+"/"+data.index
			stdout.subscribe(path, function(out){
				angular.element(el).append(lineFormatter.format(out));
			});
		}).
		error(function (error) {
			console.log(error);
		});



	$scope.rebuild = function() {
		$http({method: 'POST', url: '/v1/repos/'+remote+'/'+owner+'/'+name+'/'+'branches/'+branch+'/'+'commits/'+commit+'/builds/'+build+'?action=rebuild' })
	}


});