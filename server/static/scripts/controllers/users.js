(function () {

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
			$scope.tokens = payload.data;
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
		// Gets the currently authenticated user 
		users.getCached().then(function(payload){
			$scope.user = payload.data;
		});

		users.list().then(function(payload){
			$scope.users = payload.data;
		});

		$scope.login="";
		$scope.add = function(login) {
			users.post(login).then(function(payload){
				$scope.users.push(payload.data);
				$scope.login="";
			});
		}

		$scope.toggle = function(user) {
			user.admin = !user.admin;
			users.put(user);
		}

		$scope.remove = function(user) {
			users.delete(user).then(function(){
				var index = $scope.users.indexOf(user);
				$scope.users.splice(index, 1);
			});
		}
	}

	angular
		.module('drone')
		.controller('UserCtrl', UserCtrl)
		.controller('UsersCtrl', UsersCtrl);
})();