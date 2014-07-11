'use strict';

var app = angular.module('app', [
  'ngRoute',
  'ui.filters'
]);

app.config(['$routeProvider', '$locationProvider', function($routeProvider, $locationProvider) {
	$routeProvider.when('/', {
			templateUrl: '/views/home.html',
			controller: 'HomeController',
			title: 'Dashboard',
			resolve: {
				user: function(authService) {
					return authService.getUser();
				}
			}
		})
		.when('/sync', {
			templateUrl: '/views/sync.html',
			controller: 'SyncController',
			title: 'Sync'
		})
		.when('/login', {
			templateUrl: '/views/login.html',
			title: 'Login',
		})
		.when('/setup', {
			templateUrl: '/views/setup.html',
			controller: 'SetupController',
			title: 'Setup'
		})
		.when('/setup/:remote', {
			templateUrl: '/views/setup.html',
			controller: 'SetupController',
			title: 'Setup'
		})
		.when('/account/profile', {
			templateUrl: '/views/account.html',
			controller: 'UserController',
			title: 'Profile',
			resolve: {
				user: function(authService) {
					return authService.getUser();
				}
			}
		})
		.when('/account/repos', {
			templateUrl: '/views/repo_list.html',
			controller: 'AccountReposController',
			title: 'Repositories',
			resolve: {
				user: function(authService) {
					return authService.getUser();
				}
			}
		})
		.when('/admin/users/add', {
			templateUrl: '/views/users_add.html',
			controller: 'UserAddController',
			title: 'Add User',
			resolve: {
				user: function(authService) {
					return authService.getUser();
				}
			}
		})
		.when('/admin/users/:host/:login', {
			templateUrl: '/views/users_edit.html',
			controller: 'UserEditController',
			title: 'Edit User',
			resolve: {
				user: function(authService) {
					return authService.getUser();
				}
			}
		})
		.when('/admin/users', {
			templateUrl: '/views/users.html',
			controller: 'UsersController',
			title: 'System Users',
			resolve: {
				user: function(authService) {
					return authService.getUser();
				}
			}
		})
		.when('/admin/settings', {
			templateUrl: '/views/sys_config.html',
			controller: 'ConfigController',
			title: 'System Settings',
			resolve: {
				user: function(authService) {
					return authService.getUser();
				},
				conf: function(confService) {
					return confService.getConfig();
				}
			}
		})
		.when('/:remote/:owner/:name/settings', {
			templateUrl: '/views/repo_edit.html',
			controller: 'RepoConfigController',
			title: 'Repository Settings',
			resolve: {
				user: function(authService) {
					return authService.getUser();
				}
			}
		})
		.when('/:remote/:owner/:name/:branch/:commit', {
			templateUrl: '/views/commit.html',
			controller: 'CommitController',
			title: 'Recent Commits',
			resolve: {
				user: function(authService) {
					return authService.getUser();
				}
			}
		})
		.when('/:remote/:owner/:name', {
			templateUrl: '/views/repo.html',
			controller: 'RepoController',
			title: 'Recent Commits',
			resolve: {
				user: function(authService) {
					return authService.getUser();
				},
				repo: function($route, repos) {
					var remote = $route.current.params.remote;
					var owner  = $route.current.params.owner;
					var name   = $route.current.params.name;
					return repos.getRepo(remote, owner, name);
				}
			}
		});

	// use the HTML5 History API
	$locationProvider.html5Mode(true);
}]);

/* also see https://coderwall.com/p/vcfo4q */
app.run(['$location', '$rootScope', '$routeParams', 'feed', 'stdout', function($location, $rootScope, $routeParams, feed, stdout) {

	$rootScope.$on('$routeChangeStart', function (event, next) {
		feed.unsubscribe();
		stdout.unsubscribe();
	});

    $rootScope.$on('$routeChangeSuccess', function (event, current, previous) {
        document.title = current.$$route.title + ' · drone.io';
    });
}]);




/* Controllers */




app.controller("AccountReposController", function($scope, $http, user) {

	$scope.user = user;

	// get the user details
	$http({method: 'GET', url: '/v1/user/repos'}).
		success(function(data, status, headers, config) {
			$scope.repos = (typeof data==='string')?[]:data;
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});

	$scope.active="";
	$scope.remote="";
	$scope.byActive = function(entry){
			switch (true) {
			case $scope.active == "true"  && !entry.active: return false;
			case $scope.active == "false" &&  entry.active: return false;
			}
			return true;
		};
	$scope.byRemote = function(entry){
			return $scope.remote == "" || $scope.remote == entry.remote; 
		}; 
});



app.controller("ConfigController", function($scope, $http, user) {

	$scope.user = user;

	$http({method: 'GET', url: '/v1/config'}).
		success(function(data, status, headers, config) {
			$scope.config = data;
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});
});





app.controller("CommitController", function($scope, $http, $routeParams, stdout, feed, notify) {

	var remote = $routeParams.remote;
	var owner  = $routeParams.owner;
	var name   = $routeParams.name;
	var branch = $routeParams.branch;
	var commit = $routeParams.commit;

	feed.subscribe(function(item) {
		// if the
		if (item.commit.sha == commit
				&& item.commit.branch == branch) {
			$scope.commit = item.commit;
			$scope.$apply();
		} else {
			// we trigger an html5 notification so the
			// user is aware another build started
			notify.sendCommit(
				item.repo,
				item.commit
			);
		}
	});

	// load the repo meta-data
	$http({method: 'GET', url: '/v1/repos/'+remote+'/'+owner+"/"+name}).
		success(function(data, status, headers, config) {
			$scope.repo = data;
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});

	// load the repo commit data
	$http({method: 'GET', url: '/v1/repos/'+remote+'/'+owner+"/"+name+"/branches/"+branch+"/commits/"+commit}).
		success(function(data, status, headers, config) {
			$scope.commit = data;

			if (data.status!='Started' && data.status!='Pending') {
				return;
			}

			stdout.subscribe(data.id, function(out){
				console.log(out);
			});
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});

	// load the repo build stdout
	$http({method: 'GET', url: '/v1/repos/'+remote+'/'+owner+"/"+name+"/branches/"+branch+"/commits/"+commit+"/console"}).
		success(function(data, status, headers, config) {
			$scope.console = data;
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});

});