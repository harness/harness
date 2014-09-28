'use strict';

app.controller("CommitController", function($scope, $http, $routeParams, builds, feed) {
	var remote = $routeParams.remote;
	var owner  = $routeParams.owner;
	var name   = $routeParams.name;
	var branch = $routeParams.branch;
	var commit = $routeParams.commit;

	feed.subscribe(function(item) {
		if (item.commit.sha    == commit &&
			item.commit.branch == branch) {
			$scope.builds = item.builds;
			$scope.$apply();
		} else {
			// we trigger an toast notification so the
			// user is aware another build started
			
		}
	});

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

	builds.feed(remote, owner, name, branch, commit).success(function (feed) {
			$scope.builds = (typeof feed==='string')?[]:feed;
			$scope.state = 1;
		})
		.error(function (error) {
			$scope.builds = undefined;
			$scope.state = 1;
		});

	$scope.rebuild = function() {
		$http({method: 'POST', url: '/v1/repos/'+remote+'/'+owner+'/'+name+'/'+'branches/'+branch+'/'+'commits/'+commit+'?action=rebuild' })
	}


});