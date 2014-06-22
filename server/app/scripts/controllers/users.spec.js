'use strict';

describe('users controller', function(){
	var $controller, createController, $scope, httpBackend;
	$scope = {};

	beforeEach(module('app'));

	beforeEach(inject(function (_$controller_, $httpBackend) {
		$controller = _$controller_;
		httpBackend = $httpBackend;
	}));

	createController = function () {
		return $controller('UsersController', {
			'$scope': $scope,
			'user': {}
		});
	};

	afterEach(function () {
		httpBackend.verifyNoOutstandingRequest();
		httpBackend.verifyNoOutstandingExpectation();
	});

	it('should get the list of users', function(){
		var controller, users;
		users = [
			'brad',
			'nathan',
			'solomon'
		];
		$scope.users = [];
		httpBackend.whenGET('/v1/users').respond(200, users);
		controller = createController();
		httpBackend.flush();
		expect($scope.users).toEqual(users);
	});
});