'use strict';

(function () {

	/**
	 * Creates the angular application.
	 */
	angular.module('drone', [
			'ngRoute',
			'ui.filters'
		]);

	/**
	 * Bootstraps the application and retrieves the
	 * token from the
	 */
	function Authorize() {
		// First, parse the query string
		var params = {}, queryString = location.hash.substring(1),
			regex  = /([^&=]+)=([^&]*)/g, m;

		// Loop through and retrieve the token
		while (m = regex.exec(queryString)) {
			params[decodeURIComponent(m[1])] = decodeURIComponent(m[2]);
		}

		// if the user has just received an auth token we
		// should extract from the URL, save to local storage
		// and then remove from the URL for good measure.
		if (params.access_token) {
			localStorage.setItem("access_token", params.access_token);
			history.replaceState({}, document.title, location.pathname);
		}
	}

	/**
	 * Defines the route configuration for the
	 * main application.
	 */
	function Config ($routeProvider, $httpProvider, $locationProvider) {

		// Resolver that will attempt to load the currently
		// authenticated user prior to loading the page.
		var resolveUser = {
			user: function(users) {
				return users.getCached();
			}
		}

		$routeProvider
		.when('/', {
			templateUrl: '/static/scripts/views/repos.html',
			controller: 'ReposCtrl',
			resolve: resolveUser
		})
		.when('/login', {
			templateUrl: '/static/scripts/views/login.html',
			controller: 'UserLoginCtrl'
		})
		.when('/profile', {
			templateUrl: '/static/scripts/views/user.html',
			controller: 'UserCtrl',
			resolve: resolveUser
		})
		.when('/users', {
			templateUrl: '/static/scripts/views/users.html',
			controller: 'UsersCtrl',
			resolve: resolveUser
		})
		.when('/new', {
			templateUrl: '/static/scripts/views/repos_add.html',
			controller: 'RepoAddCtrl',
			resolve: resolveUser
		})
		.when('/:owner/:name', {
			templateUrl: '/static/scripts/views/builds.html',
			controller: 'BuildsCtrl',
			resolve: resolveUser
		})
		.when('/:owner/:name/edit', {
			templateUrl: '/static/scripts/views/repos_edit.html',
			controller: 'RepoEditCtrl',
			resolve: resolveUser
		})
		.when('/:owner/:name/edit/env', {
			templateUrl: '/static/scripts/views/repos_env.html',
			controller: 'RepoEditCtrl',
			resolve: resolveUser
		})
		.when('/:owner/:name/delete', {
			templateUrl: '/static/scripts/views/repos_del.html',
			controller: 'RepoEditCtrl',
			resolve: resolveUser
		})
		.when('/:owner/:name/:number', {
			templateUrl: '/static/scripts/views/build.html',
			controller: 'BuildCtrl',
			resolve: resolveUser
		})
		.when('/:owner/:name/:number/:step', {
			templateUrl: '/static/scripts/views/build_out.html',
			controller: 'BuildOutCtrl',
			resolve: resolveUser
		});

		// Enables html5 mode
		$locationProvider.html5Mode(true)

		// Appends the Bearer token to authorize every
		// outbound http request.
		$httpProvider.defaults.headers.common.Authorization = 'Bearer '+localStorage.getItem('access_token');

		// Intercepts every oubput http response and redirects
		// the user to the logic screen if the request was rejected.
		$httpProvider.interceptors.push(function($q, $location) {
			return {
				'responseError': function(rejection) {
					if (rejection.status === 401 && rejection.config.url !== "/api/user") {
						$location.path('/login');
					}
					if (rejection.status === 0) {
						// this happens when the app is down or
						// the browser loses internet connectivity.
					}
					return $q.reject(rejection);
				}
			};
		});
	}


	function RouteChange($rootScope, repos, logs) {
		$rootScope.$on('$routeChangeStart', function (event, next) {
			repos.unsubscribe();
			logs.unsubscribe();
		});

		$rootScope.$on('$routeChangeSuccess', function (event, current, previous) {
			if (current.$$route.title) {
 				document.title = current.$$route.title + ' · drone';
			}
		});
	}

	angular
		.module('drone')
		.config(Authorize)
		.config(Config)
		.run(RouteChange);

})();
