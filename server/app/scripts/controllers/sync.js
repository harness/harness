'use strict';

angular.module('app').controller("SyncController", function($scope, $http, $interval, $location, $routeParams, users) {
	var return_to = $routeParams.return_to
	var stop = $interval(function() {
		// todo(bradrydzewski) We should poll the user to see if the
		// sync process is complete, using the user.syncing variable.
		$interval.cancel(stop);
		if (return_to != undefined) {
			$location.$$search = {}
			$location.path(return_to);
		} else {
			$location.path("/");
		}
	}, 5000);
});
