(function () {

	function UserLoginCtrl($scope, $window) {
		// attempts to extract an error message from
		// the URL hash in format #error=?
		$scope.error = $window.location.hash.substr(7);
	}

	/**
	 * UserCtrl is responsible for managing user settings.
	 */
	function UserCtrl($scope, users, tokens) {

		// Gets the currently authenticated user
		users.getCurrent().then(function(payload){
			$scope.user = payload.data;
		});

		// Gets the user tokens
		tokens.list().then(function(payload){
			$scope.tokens = payload.data || [];
		});

		$scope.newToken={Label: ""};
		$scope.createToken = function(newToken) {
			tokens.post(newToken).then(function(payload) {
				$scope.tokens.push(payload.data);
				$scope.newToken={Label: ""};
			});
		}

		$scope.revokeToken = function(token) {
			tokens.delete(token).then(function() {
				var index = $scope.tokens.indexOf(token);
				$scope.tokens.splice(index, 1);
			});
		}
	}

	/**
	 * UsersCtrl is responsible for managing user accounts.
	 * This part of the site is for administrators only.
	 */
	function UsersCtrl($scope, users) {
	    $scope.loading = true;
	    $scope.waiting = false;

		// Gets the currently authenticated user
		users.getCached().then(function(payload){
			$scope.user = payload.data;
		});

		// Gets the list of all system users
		users.list().then(function(payload){
	    	$scope.loading = true;
			$scope.users = payload.data;
		});

		$scope.add = function(event, login) {
			$scope.error = undefined;
			$scope.new_user = undefined;
			if (event.which && event.which !== 13) {
				return;
			}
			$scope.waiting = true;

			users.post(login).then(function(payload){
				$scope.users.push(payload.data);
				$scope.search_text=undefined;
				$scope.waiting = false;
				$scope.new_user = payload.data;
			}).catch(function (err) {
				$scope.error = err;
				$scope.waiting = false;
				$scope.search_text = undefined;
			});
		}

		$scope.toggle = function(user) {
			if (user.login === $scope.user.login) {
				// cannot revoke admin privilege for self
				$scope.error = {}; // todo display an actual error here
				return;
			}
			user.admin = !user.admin;
			users.put(user);
		}

		$scope.remove = function(user) {
			if (user.login === $scope.user.login) {
				// cannot delete self
				$scope.error = {}; // todo display an actual error here
				return;
			}
			users.delete(user).then(function(){
				var index = $scope.users.indexOf(user);
				$scope.users.splice(index, 1);
			});
		}
	}

	angular
		.module('drone')
		.controller('UserLoginCtrl', UserLoginCtrl)
		.controller('UserCtrl', UserCtrl)
		.controller('UsersCtrl', UsersCtrl);
})();
