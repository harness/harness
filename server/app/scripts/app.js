'use strict';

var app = angular.module('app', [
  'ngRoute',
  'ui.filters',
  'angularMoment'
]);

angular.module('app').constant('angularMomentConfig', {
    preprocess: 'unix'
});

// First, parse the query string
var params = {}, queryString = location.hash.substring(1),
	regex  = /([^&=]+)=([^&]*)/g, m;
while (m = regex.exec(queryString)) {
	params[decodeURIComponent(m[1])] = decodeURIComponent(m[2]);
}


// if the user is authenticated we should add Basic
// auth token to each request.
if (params.access_token) {
	localStorage.setItem("access_token", params.access_token);
	history.replaceState({}, document.title, location.pathname);
}



app.config(['$routeProvider', '$locationProvider', '$httpProvider', function($routeProvider, $locationProvider, $httpProvider) {
	$routeProvider.when('/', {
			templateUrl: '/static/views/home.html',
			controller: 'HomeController',
			title: 'Dashboard',
			resolve: {
				user: function(authService) {
					return authService.getUser();
				}
			}
		})
		.when('/sync', {
			templateUrl: '/static/views/sync.html',
			controller: 'SyncController',
			title: 'Sync'
		})
		.when('/login', {
			templateUrl: '/static/views/login.html',
			controller: 'LoginController',
			title: 'Login'
		})
		.when('/logout', {
			templateUrl: '/static/views/logout.html',
			controller: 'LogoutController',
			title: 'Logout'
		})
		.when('/gitlab', {
			templateUrl: '/static/views/login_gitlab.html',
			title: 'GitLab Login',
		})
		.when('/gogs', {
			templateUrl: '/static/views/login_gogs.html',
			title: 'Gogs Setup',
		})
		.when('/setup', {
			templateUrl: '/static/views/setup.html',
			controller: 'SetupController',
			title: 'Setup'
		})
		.when('/setup/:remote', {
			templateUrl: '/static/views/setup.html',
			controller: 'SetupController',
			title: 'Setup'
		})
		.when('/account/profile', {
			templateUrl: '/static/views/account.html',
			controller: 'UserController',
			title: 'Profile',
			resolve: {
				user: function(authService) {
					return authService.getUser();
				}
			}
		})
		.when('/account/repos', {
			templateUrl: '/static/views/repo_list.html',
			controller: 'AccountReposController',
			title: 'Repositories',
			resolve: {
				user: function(authService) {
					return authService.getUser();
				}
			}
		})
		.when('/admin/users/add', {
			templateUrl: '/static/views/users_add.html',
			controller: 'UserAddController',
			title: 'Add User',
			resolve: {
				user: function(authService) {
					return authService.getUser();
				}
			}
		})
		.when('/admin/users/:host/:login', {
			templateUrl: '/static/views/users_edit.html',
			controller: 'UserEditController',
			title: 'Edit User',
			resolve: {
				user: function(authService) {
					return authService.getUser();
				}
			}
		})
		.when('/admin/users', {
			templateUrl: '/static/views/users.html',
			controller: 'UsersController',
			title: 'System Users',
			resolve: {
				user: function(authService) {
					return authService.getUser();
				}
			}
		})
		.when('/admin/settings', {
			templateUrl: '/static/views/config.html',
			controller: 'ConfigController',
			title: 'System Settings',
			resolve: {
				user: function(authService) {
					return authService.getUser();
				}
			}
		})
		.when('/:remote/:owner/:name/settings', {
			templateUrl: '/static/views/repo_edit.html',
			controller: 'RepoConfigController',
			title: 'Repository Settings',
			resolve: {
				user: function(authService) {
					return authService.getUser();
				}
			}
		})
		.when('/:remote/:owner/:name/:branch*\/:commit', {
			templateUrl: '/static/views/commit.html',
			controller: 'CommitController',
			title: 'Recent Commits',
			resolve: {
				user: function(authService) {
					return authService.getUser();
				}
			}
		})
		.when('/:remote/:owner/:name', {
			templateUrl: '/static/views/repo.html',
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

	$httpProvider.defaults.headers.common.Authorization = 'Bearer '+localStorage.getItem('access_token');

	$httpProvider.interceptors.push(function($q, $location) {
		return {
			'responseError': function(rejection) {
				if (rejection.status == 401 && rejection.config.url != "/api/user") {
					$location.path('/login');
				}
				return $q.reject(rejection);
			}
		};
	});
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




app.controller("AccountReposController", function($scope, $http, $location, user) {

	$scope.user = user;

	$scope.syncUser = function() {
		$http({method: 'POST', url: '/api/user/sync' }).success(function(data){
			$location.search('return_to', $location.$$path).path('/sync')
		}).error(function(data, status){
			if (status == 409) {
				$scope.msg = 'already'
			} else {
				$scope.msg = 'bad'
			}
			$scope.$apply();
		});
	}

	// get the user details
	$http({method: 'GET', url: '/api/user/repos'}).
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
