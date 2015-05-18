(function () {

	function AgentsCtrl($scope, $window, users, agents) {
		// this is the address that agents should connect with.
		$scope.addr = $window.location.origin;
		
		// Gets the currently authenticated user
		users.getCached().then(function(payload){
			$scope.user = payload.data;
		});

		// Generages a remote agents.
		agents.getAgents().then(function(payload){
			$scope.agents = payload.data;
		});

		$scope.onDelete = function(agent) {
			console.log("delete agent", agent)
			agents.deleteAgent(agent).then(function(payload){
				var index = $scope.agents.indexOf(agent);
				$scope.agents.splice(index, 1);
			});
		}

		$scope.newAgent={address: ""};
		$scope.onAdd = function(agent) {
			agents.postAgent(agent).then(function(payload){
				$scope.agents.push(payload.data);
				$scope.newAgent={address: ""};
			});
		}
	}

	angular
		.module('drone')
		.controller('AgentsCtrl', AgentsCtrl);
})();
