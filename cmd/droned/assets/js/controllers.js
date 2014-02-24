"use strict";

/* Controllers */

var ctlMod = angular.module( "drone.controllers", [] );

ctlMod.controller( "Projects", [ "$scope", "$rootScope", "$http", function ( $scope, $rootScope, $http )
{
    var url = window.location.pathname + "/commits"

    ws.onopen = function(){  
        console.log("Socket has been opened!");  
    };
    
    ws.onmessage = function(message) {
        var payload = {};
        var task = JSON.parse(message.data);

        payload.owner        = task.Repo.owner 
        payload.name         = task.Repo.name
        payload.hash         = task.Commit.hash;
        payload.status       = task.Commit.status;
        payload.created      = task.Commit.created;
        payload.pull_request = task.Commit.pull_request;
        payload.gravatar     = task.Commit.gravatar;
        payload.message      = task.Commit.message;

        $scope.addBuild(payload)
    };

	$http.get( url ).success( function ( result )
	{
		$scope.projects = [];
		
		$scope.addBuild = function ( newBuild )
		{
			currentProject = '';
			
			for( var j = 0; j < $scope.projects.length; j++ )
			{
				if( $scope.projects[j].projectOwner == newBuild.owner && $scope.projects[j].projectName == newBuild.name )
				{
					currentProject = $scope.projects[j];
					break;
				}
			}
			
			if( !currentProject )
			{
				currentProject = {};
				currentProject.projectOwner = newBuild.owner;
				currentProject.projectName  = newBuild.name;
				currentProject.builds       = [];
				
				$scope.projects.push( currentProject );
				currentProject = $scope.projects[ $scope.projects.length - 1 ];
			}
	
			build = {};
			build.fresh = true;
			
			for( var k = 0; k < currentProject.builds.length; k++ )
			{
				if( currentProject.builds[k].hash == newBuild.hash )
				{
					build       = currentProject.builds[k];
					build.fresh = false;
					break;
				}
			}
			
			build.hash      = newBuild.hash;
			build.status    = newBuild.status;
			build.buildTime = newBuild.created;
			build.pull      = newBuild.pull_request;
			build.gravatar  = newBuild.gravatar;
			build.message   = newBuild.message;
			
			currentProject.masterHash   = !currentProject.masterHash   && !build.pull ? build.hash   : currentProject.masterHash;
			currentProject.masterStatus = !currentProject.masterStatus && !build.pull ? build.status : currentProject.masterStatus;
			
			if( build.fresh )
			{
				currentProject.builds.push( build );
			}
		};
		
		for( var i = 0; i < result.length; i++ )
		{
			$scope.addBuild( result[i] );
		}
		
	} ).error( function ()
	{
		console.log( "Couldn't connect to drone, great job." );
		
	} );

} ] );
