'use strict';

angular.module('app').controller("LogoutController", function() {
	console.log("logging out")
	localStorage.removeItem("access_token");
	window.location.href = "/login";
});