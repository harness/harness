'use strict';

(function () {

	/**
	 * Cached user object.
	 */
	var _user;

	/**
	 * The UserService provides access to useer
	 * data using REST API calls.
	 */
	function UserService($http, $q) {

		/**
		 * Gets a list of all users.
		 */
		this.list = function() {
			return $http.get('api/users');
		};

		/**
		 * Gets a user by login.
		 */
		this.get = function(login) {
			return $http.get('api/users/'+login);
		};

		/**
		 * Gets the currently authenticated user.
		 */
		this.getCurrent = function() {
			return $http.get('api/user');
		};

		/**
		 * Updates an existing user
		 */
		this.post = function(user) {
			return $http.post('api/users/'+user);
		};

		/**
		 * Updates an existing user
		 */
		this.put = function(user) {
			return $http.patch('api/users/'+user.login, user);
		};

		/**
		 * Deletes a user.
		 */
		this.delete = function(user) {
			return $http.delete('api/users/'+user.login);
		};

		/**
		 * Gets the currently authenticated user from
		 * the local cache. If not exists, it will fetch
		 * from the server.
		 */
		this.getCached = function() {
			var defer = $q.defer();

			// if the user is already authenticated
			if (_user) {
				defer.resolve(_user);
				return defer.promise;
			}

			// else fetch the currently authenticated
			// user using the REST API.
			this.getCurrent().then(function(payload){
				_user=payload;
				defer.resolve(_user);
			}).catch(function(){
				defer.resolve(_user);
			});

			return defer.promise;
		}
	}

	angular
		.module('drone')
		.service('users', UserService);
})();
