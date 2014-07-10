'use strict';

angular.module('app').controller("SyncController", function($scope, $http, $interval, $location, users) {
	$interval(function() {
		// todo(bradrydzewski) We should poll the user to see
		// if the sync process is complete. If no, we should
		// repeat.
		$location.path("/");
	}, 5000);
});
