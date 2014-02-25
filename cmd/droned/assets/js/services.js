"use strict";

/* Services */

// Demonstrate how to register services
// In this case it is a simple value service.

var svcMod = angular.module( "drone.services", [] );

svcMod.value( "version", "0.1" );
