(function () {

	function AgentsCtrl($scope, $window, users, agents) {

    // this is the address that agents should connect with.
    $scope.addr = $window.location.origin;

    // Gets the currently authenticated user
		users.getCached().then(function(payload){
			$scope.user = payload.data;
		});

		// Generages a remote token.
		agents.getToken().then(function(payload){
			$scope.token = payload.data;
		});
	}

	angular
		.module('drone')
		.controller('AgentsCtrl', AgentsCtrl);
})();
