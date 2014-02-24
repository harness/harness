"use strict";

// Declare app level module which depends on filters, and services
var angMod = angular.module( "drone", [
	"drone.filters",
	"drone.services",
	"drone.directives",
	"drone.controllers"
], function($interpolateProvider) {
    $interpolateProvider.startSymbol('[[');
    $interpolateProvider.endSymbol(']]');
} );
