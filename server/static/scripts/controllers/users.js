(function () {

	/**
	 * UserCtrl is responsible for managing user settings.
	 */	
	function UserCtrl($scope, users) {

		// Gets the currently authenticated user 
		users.getCurrent().then(function(payload){
			$scope.user = payload.data;
		});
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
				users.list().then(function(payload){
					$scope.users = payload.data;
				});
			});
		}
	}

	angular
		.module('drone')
		.controller('UserCtrl', UserCtrl)
		.controller('UsersCtrl', UsersCtrl);
})();