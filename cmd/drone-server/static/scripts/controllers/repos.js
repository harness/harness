(function () {

  /**
   * ReposCtrl responsible for rendering the user's
   * repository home screen.
   */
  function ReposCtrl($scope, $location, $stateParams, repos, users) {
    $scope.loading = true;
    $scope.waiting = false;

    // Gets the currently authenticated user
    users.getCached().then(function (payload) {
      $scope.user = payload.data;
      $scope.loading = false;
    });

    // Gets a list of repos to display in the
    // dropdown.
    repos.list().then(function (payload) {
      $scope.repos = angular.isArray(payload.data) ? payload.data : [];
    }).catch(function (err) {
      $scope.error = err;
    });

    // Adds a repository
    $scope.add = function (event, fullName) {
      $scope.error = undefined;
      if (event.which && event.which !== 13) {
        return;
      }
      $scope.waiting = true;

      repos.post(fullName).then(function (payload) {
        $location.path('/' + fullName);
        $scope.waiting = false;
      }).catch(function (err) {
        $scope.error = err;
        $scope.waiting = false;
        $scope.search_text = undefined;
      });
    }
  }

  /**
   * RepoAddCtrl responsible for activaing a new
   * repository.
   */
  function RepoAddCtrl($scope, $location, repos, users) {

    // Gets the currently authenticated user
    users.getCached().then(function (payload) {
      $scope.user = payload.data;
    });

    $scope.add = function (slug) {
      repos.post(slug).then(function (payload) {
        $location.path('/' + slug);
      }).catch(function (err) {
        $scope.error = err;
      });
    }
  }

  /**
   * RepoEditCtrl responsible for editing a repository.
   */
  function RepoEditCtrl($scope, $window, $location, $stateParams, repos, users) {
    var owner = $stateParams.owner;
    var name = $stateParams.name;
    var fullName = owner + '/' + name;

    // Inject window for composing url
    $scope.window = $window;

    // Gets the currently authenticated user
    users.getCached().then(function (payload) {
      $scope.user = payload.data;
    });

    // Gets a repository
    repos.get(fullName).then(function (payload) {
      $scope.repo = payload.data;
    }).catch(function (err) {
      $scope.error = err;
    });

    $scope.save = function (repo) {
      repo.timeout = parseInt(repo.timeout);
      repos.update(repo).then(function (payload) {
        $scope.repo = payload.data;
      }).catch(function (err) {
        $scope.error = err;
      });
    };

    $scope.delete = function (repo) {
      repos.delete(repo).then(function (payload) {
        $location.path('/');
      }).catch(function (err) {
        $scope.error = err;
      });
    };

    $scope.param = {};
    $scope.addParam = function (param) {
      if (!$scope.repo.params) {
        $scope.repo.params = {}
      }
      $scope.repo.params[param.key] = param.value;
      $scope.param = {};

      // auto-update
      repos.update($scope.repo).then(function (payload) {
        $scope.repo = payload.data;
      }).catch(function (err) {
        $scope.error = err;
      });
    };


    $scope.encrypt = function (plaintext) {
      repos.encrypt(fullName, plaintext).then(function (payload) {
        $scope.secure = payload.data;
      }).catch(function (err) {
        $scope.error = err;
      });
    };

    $scope.deleteParam = function (key) {
      delete $scope.repo.params[key];

      // auto-update
      repos.update($scope.repo).then(function (payload) {
        $scope.repo = payload.data;
      }).catch(function (err) {
        $scope.error = err;
      });
    }
  }

  function toSnakeCase(str) {
    return str.replace(/ /g, '_').replace(/([a-z0-9])([A-Z0-9])/g, '$1_$2').toLowerCase();
  }

  angular
    .module('drone')
    .controller('ReposCtrl', ReposCtrl)
    .controller('RepoAddCtrl', RepoAddCtrl)
    .controller('RepoEditCtrl', RepoEditCtrl);
})();
