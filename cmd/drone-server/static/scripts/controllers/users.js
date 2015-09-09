(function () {

	function UserHeaderCtrl($scope, $stateParams, users) {
		// Gets the currently authenticated user
		users.getCurrent().then(function(payload){
			$scope.user = payload.data;
		});

		$scope.number = $stateParams.number || undefined;
		$scope.owner = $stateParams.owner || undefined;
		$scope.name = $stateParams.name || undefined;
		$scope.full_name = $scope.owner + '/' + $scope.name;
	}

	function UserLoginCtrl($scope, $window) {
		// attempts to extract an error message from
		// the URL hash in format #error=?
		$scope.error = $window.location.hash.substr(7);
	}

	function UserLogoutCtrl($scope, $window, $state) {
		// Remove login information from the local
		// storage and redirect to login page
		if (localStorage.hasOwnProperty("access_token")) {
			localStorage.removeItem("access_token");
		}

		$state.go("login", {}, {
			location: "replace"
		});
	}

	/**
	 * UserCtrl is responsible for managing user settings.
	 */
	function UserCtrl($scope, users, tokens) {

		// Gets the currently authenticated user
		users.getCurrent().then(function(payload){
			$scope.user = payload.data;
		});

		$scope.showToken = function() {
			tokens.post().then(function(payload) {
				$scope.token = payload.data;
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
		.controller('UserHeaderCtrl', UserHeaderCtrl)
		.controller('UserLoginCtrl', UserLoginCtrl)
		.controller('UserLogoutCtrl', UserLogoutCtrl)
		.controller('UserCtrl', UserCtrl)
		.controller('UsersCtrl', UsersCtrl);
})();
