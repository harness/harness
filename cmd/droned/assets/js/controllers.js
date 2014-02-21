"use strict";

/* Controllers */

var ctlMod = angular.module( "drone.controllers", [] );

ctlMod.controller( "Projects", [ "$scope", "$rootScope", "$http", function ( $scope, $rootScope, $http )
{
	$http.get( "/drone/url/path" ).success( function ( result )
	{
		$scope.projects = [];
		var currentProject, build;
		
		for( var i = 0; i < result.length; i++ )
		{
			currentProject = '';
			
			for( var j = 0; j < $scope.projects.length; j++ )
			{
				if( $scope.projects[j].projectOwner == result[i].owner && $scope.projects[j].projectName == result[i].name )
				{
					currentProject = $scope.projects[j];
					break;
				}
			}
			
			if( !currentProject )
			{
				currentProject = {};
				currentProject.projectOwner = result[i].owner;
				currentProject.projectName  = result[i].name;
				currentProject.builds       = [];
				
				$scope.projects.push( currentProject );
				currentProject = $scope.projects[ $scope.projects.length - 1 ];
			}
			
			build = {};
			build.hash      = result[i].hash;
			build.status    = result[i].status;
			build.buildTime = result[i].created;
			build.pull      = result[i].pull_request;
			build.gravatar  = result[i].gravatar;
			build.message   = result[i].message;
			
			currentProject.masterHash   = !currentProject.masterHash   && !build.pull ? build.hash   : currentProject.masterHash;
			currentProject.masterStatus = !currentProject.masterStatus && !build.pull ? build.status : currentProject.masterStatus;
			
			currentProject.builds.push( build );
			
		}
		
	} ).error( function ()
	{
		console.log( "Couldn't connect to drone, great job." );
		
	} );

} ] );