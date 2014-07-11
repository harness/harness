'use strict';

angular.module('app').controller("SyncController", function($scope, $http, $interval, $location, users) {
	var stop = $interval(function() {
		// todo(bradrydzewski) We should poll the user to see
		// if the sync process is complete.
		$interval.cancel(stop);
		$location.path("/");
	}, 5000);
});
