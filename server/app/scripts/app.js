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
			templateUrl: '/views/repo_conf.html',
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
		.when('/:remote/:owner/:name/:branch', {
			templateUrl: '/views/branch.html',
			controller: 'BranchController',
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
				repo: function($route, repoService) {
					var remote = $route.current.params.remote;
					var owner  = $route.current.params.owner;
					var name   = $route.current.params.name;
					return repoService.getRepo(remote, owner, name);
				}
			}
		});

	// use the HTML5 History API
	$locationProvider.html5Mode(true);
}]);

/* Directives */

/* also see https://coderwall.com/p/vcfo4q */
app.run(['$location', '$rootScope', '$routeParams', function($location, $rootScope, $routeParams) {
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

app.controller("RepoConfigController", function($scope, $http, $routeParams, user) {
	$scope.user = user;

	var remote = $routeParams.remote;
	var owner  = $routeParams.owner;
	var name   = $routeParams.name;

	// load the repo meta-data
	$http({method: 'GET', url: '/v1/repos/'+remote+'/'+owner+"/"+name+"?admin=1"}).
		success(function(data, status, headers, config) {
			$scope.repo = data;
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});

	$scope.save = function() {
		// request to create a new repository
		$http({method: 'PUT', url: '/v1/repos/'+remote+'/'+owner+"/"+name, data: $scope.repo }).
			success(function(data, status, headers, config) {
				delete $scope.failure;
			}).
			error(function(data, status, headers, config) {
				$scope.failure = data;
			});
	};
});

function badgeMarkdown(repo) {
	var scheme = window.location.protocol;
	var host = window.location.host;
	return '[![Build Status]('+scheme+'//'+host+'/v1/badge/'+repo+'/status.svg?branch=master)]('+scheme+'//'+host+'/'+repo+')'
}

function badgeMarkup(repo) {
	var scheme = window.location.protocol;
	var host = window.location.host;
	return '<a href="'+scheme+'//'+host+'/'+repo+'"><img src="'+scheme+'//'+host+'/v1/badge/'+repo+'/status.svg?branch=master" /></a>'
}

app.controller("RepoController", function($scope, $http, $routeParams, user, repo) {
	$scope.user = user;
	$scope.repo = repo;

	// load the repo branch list
	/*
	$http({method: 'GET', url: '/v1/repos/'+repo.host+'/'+repo.owner+"/"+repo.name+"/branches"}).
		success(function(data, status, headers, config) {
			$scope.branches = (typeof data==='string')?[]:data;
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});
	*/

	// load the repo commit feed
	$http({method: 'GET', url: '/v1/repos/'+repo.host+'/'+repo.owner+"/"+repo.name+"/feed"}).
		success(function(data, status, headers, config) {
			$scope.commits = (typeof data==='string')?[]:data;
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});

	$scope.activate = function() {
		// request to create a new repository
		$http({method: 'POST', url: '/v1/repos/'+repo.host+'/'+repo.owner+"/"+repo.name }).
			success(function(data, status, headers, config) {
				$scope.repo = data;
			}).
			error(function(data, status, headers, config) {
				$scope.failure = data;
			});
	};

});

app.controller("BranchController", function($scope, $http, $routeParams, user) {

	$scope.user = user;
	var remote = $routeParams.remote;
	var owner  = $routeParams.owner;
	var name   = $routeParams.name;
	var branch = $routeParams.branch;
	$scope.branch = branch;

	// load the repo meta-data
	$http({method: 'GET', url: '/v1/repos/'+remote+'/'+owner+"/"+name}).
		success(function(data, status, headers, config) {
			$scope.repo = data;
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});

	// load the repo branch list
	$http({method: 'GET', url: '/v1/repos/'+remote+'/'+owner+"/"+name+"/branches"}).
		success(function(data, status, headers, config) {
			$scope.branches = (typeof data==='string')?[]:data;
		}).
		error(function(data, status, headers, config) {
			console.log(data);
		});

	// load the repo commit feed
	$http({method: 'GET', url: '/v1/repos/'+remote+'/'+owner+"/"+name+"/branches/"+branch+"/commits"}).
		success(function(data, status, headers, config) {
			$scope.commits = data;
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

	// load the repo build data
	$http({method: 'GET', url: '/v1/repos/'+remote+'/'+owner+"/"+name+"/branches/"+branch+"/commits/"+commit+"/builds/1"}).
		success(function(data, status, headers, config) {
			$scope.build = data;
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