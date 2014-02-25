"use strict";

/* Directives */

var dirMod = angular.module( "drone.directives", [] );

dirMod.directive( "appVersion", [ "version", function ( version )
{
	return function ( scope, elm, attrs )
	{
		elm.text( version );
	};
} ] );
