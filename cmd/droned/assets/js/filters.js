"use strict";

/* Filters */

var fltMod = angular.module( "drone.filters", [] );

fltMod.filter( "interpolate", [ "version", function ( version )
{
	return function ( text )
	{
		return String( text ).replace( /\%VERSION\%/mg, version );
	}
} ] );
