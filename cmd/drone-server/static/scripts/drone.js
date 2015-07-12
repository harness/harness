'use strict';

(function () {

	/**
	 * Creates the angular application.
	 */
	angular.module('drone', [
			'ngRoute',
			'ui.filters',
			'ui.router'
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
	function Config ($stateProvider, $httpProvider, $locationProvider) {

		// Resolver that will attempt to load the currently
		// authenticated user prior to loading the page.
		var resolveUser = {
			user: function(users) {
				return users.getCached();
			}
		}

		$stateProvider
			.state('app', {
				abstract: true,
				views: {
					'layout': { 
						templateUrl: '/static/scripts/views/layout.html',
						controller: function ($scope, $routeParams, repos, users) {
							users.getCached().then(function(payload){
								$scope.user = payload.data;
								console.log(repos.list());
							});
						}
					}
				},
				resolve: resolveUser
			})
			.state('app.index', {
				url: '/',
				views: {
					'toolbar': { 
						templateUrl: '/static/scripts/views/repos/index/toolbar.html'
					},
					'content': { 
						templateUrl: '/static/scripts/views/repos/index/content.html',
						controller: 'ReposCtrl',
						resolve: resolveUser
					}
				},
				title: 'Dashboard'
			})
			.state('login', {
				url: '/login',
				templateUrl: '/static/scripts/views/login.html',
				title: 'Login',
				controller: 'UserLoginCtrl'
			})
			.state('app.profile', {
				url: '/profile',
				views: {
					'toolbar': { templateUrl: '/static/scripts/views/profile/toolbar.html' },
					'content': { 
						templateUrl: '/static/scripts/views/profile/content.html',
						controller: 'UserCtrl',
						resolve: resolveUser
					}
				},
				title: 'Profile'
			})
			.state('app.users', {
				url: '/users',
				views: {
					'toolbar': { templateUrl: '/static/scripts/views/users/toolbar.html' },
					'content': { 
						templateUrl: '/static/scripts/views/users/content.html',
						controller: 'UsersCtrl',
						resolve: resolveUser
					}
				},
				title: 'Users'
			})
			.state('app.new_repo', {
				url: '/new',
				views: {
					'toolbar': { templateUrl: '/static/scripts/views/repos/add/toolbar.html' },
					'content': { 
						templateUrl: '/static/scripts/views/repos/add/content.html',
						controller: 'RepoAddCtrl',
						resolve: resolveUser
					}
				},
				title: 'Add Repository'
			})
			.state('app.builds', {
				url: '/:owner/:name',
				views: {
					'toolbar': { 
						templateUrl: '/static/scripts/views/builds/index/toolbar.html',
						controller: 'BuildsCtrl'
					},
					'content': { 
						templateUrl: '/static/scripts/views/builds/index/content.html',
						controller: 'BuildsCtrl'
					}
				}
			})
			.state('app.repo_edit', {
				url: '/:owner/:name/edit',
				views: {
					'toolbar': { templateUrl: '/static/scripts/views/repos/toolbar.html' },
					'content': { templateUrl: '/static/scripts/views/repos/edit.html' }
				},
				controller: 'RepoEditCtrl',
				resolve: resolveUser
			})
			.state('app.repo.env', {
				url: '/:owner/:name/edit/env',
				views: {
					'toolbar': { templateUrl: '/static/scripts/views/repos/toolbar.html' },
					'content': { templateUrl: '/static/scripts/views/repos/env.html' }
				},
				controller: 'RepoEditCtrl',
				resolve: resolveUser
			})
			.state('app.repo.del', {
				url: '/:owner/:name/delete',
				views: {
					'toolbar': { templateUrl: '/static/scripts/views/repos/toolbar.html' },
					'content': { templateUrl: '/static/scripts/views/repos/del.html' }
				},
				controller: 'RepoEditCtrl',
				resolve: resolveUser
			})
			.state('app.build', {
				url: '/:owner/:name/:number',
				views: {
					'toolbar': { templateUrl: '/static/scripts/views/builds/show/toolbar.html' },
					'content': { templateUrl: '/static/scripts/views/builds/show/content.html' }
				},
				controller: 'BuildCtrl',
				resolve: resolveUser
			})
			.state('app.build_step', {
				url: '/:owner/:name/:number/:step',
				views: {
					'toolbar': { templateUrl: '/static/scripts/views/builds/step/toolbar.html' },
					'content': { templateUrl: '/static/scripts/views/builds/step/content.html' }
				},
				controller: 'BuildOutCtrl',
				resolve: resolveUser
			})

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
		$rootScope.$on('$stateChangeStart', function () {
			repos.unsubscribe();
			logs.unsubscribe();
		});

		$rootScope.$on('$stateChangeSuccess', function (event, current, previous) {
			if (current.title) {
 				document.title = current.title + ' · drone';
			}
		});
	}

	angular
		.module('drone')
		.config(Authorize)
		.config(Config)
		.run(RouteChange);

})();
