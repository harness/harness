(function () {

  /**
   * BuildsCtrl responsible for rendering the repo's
   * recent build history.
   */
  function BuildsCtrl($scope, $stateParams, builds, repos, users, logs) {
    var owner = $stateParams.owner;
    var name = $stateParams.name;
    var fullName = owner + '/' + name;

    $scope.loading=true;

    // Gets the currently authenticated user
    users.getCached().then(function (payload) {
      $scope.user = payload.data;
    });

    // Gets a repository
    repos.get(fullName).then(function (payload) {
      $scope.repo = payload.data;
      $scope.loading=false;
    }).catch(function (err) {
      $scope.error = err;
    });

    // Gets a list of builds
    builds.list(fullName).then(function (payload) {
      $scope.builds = angular.isArray(payload.data) ? payload.data : [];
    }).catch(function (err) {
      $scope.error = err;
    });

    $scope.watch = function (repo) {
      repos.watch(repo.full_name).then(function (payload) {
        $scope.repo.starred = true;
      });
    };

    $scope.unwatch = function (repo) {
      repos.unwatch(repo.full_name).then(function () {
        $scope.repo.starred = false;
      });
    };

    repos.subscribe(fullName, function (event) {
      var added = false;
      for (var i = 0; i < $scope.builds.length; i++) {
        var build = $scope.builds[i];
        if (event.number !== build.number) {
          continue; // ignore
        }
        // update the build status
        $scope.builds[i] = event;
        $scope.$apply();
        added = true;
      }

      if (!added) {
        $scope.builds.push(event);
        $scope.$apply();
      }
    });
  }

  /**
   * BuildCtrl responsible for rendering a build.
   */
  function BuildCtrl($scope, $stateParams, $window, logs, builds, repos, users) {
    var number = $stateParams.number;
    var owner = $stateParams.owner;
    var name = $stateParams.name;
    var fullName = owner + '/' + name;
    var step = parseInt($stateParams.step) || 1;

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

    // Gets the build
    builds.get(fullName, number).then(function (payload) {
      $scope.step = step;
      $scope.build = payload.data;
    }).catch(function (err) {
      $scope.error = err;
    });

    repos.subscribe(fullName, function (event) {
      if (event.number !== parseInt(number)) {
        return; // ignore
      }
      // update the build
      $scope.build = event;
      $scope.$apply();
    });

    $scope.restart = function () {
      builds.restart(fullName, number).then(function (payload) {
        $scope.build = payload.data;
      }).catch(function (err) {
        $scope.error = err;
      });
    };

    $scope.cancel = function () {
      builds.cancel(fullName, number).then(function (payload) {
        $scope.build = payload.data;
      }).catch(function (err) {
        $scope.error = err;
      });
    };

    $scope.tail = function () {
      tail = !tail;
    };
  }


  /**
   * BuildOutCtrl responsible for rendering a build output.
   */
  function BuildOutCtrl($scope, $stateParams, $window, logs, builds, repos, users) {

    var step = parseInt($stateParams.step) || 1;
    var number = $stateParams.number;
    var owner = $stateParams.owner;
    var name = $stateParams.name;
    var fullName = owner + '/' + name;
    var streaming = false;
    var tail = false;

    // Initiates streaming a build.
    var stream = function () {
      if (streaming) {
        return;
      }
      streaming = true;

      var convert = new Filter({stream: true, newline: false});
      var term = document.getElementById("term");
      term.innerHTML = "";

      // subscribes to the build otuput.
      logs.subscribe(fullName, number, step, function (data) {
        term.innerHTML += convert.toHtml(data.replace("\\n", "\n"));
        if (tail) {
          // scrolls to the bottom of the page if enabled
          $window.scrollTo(0, $window.document.body.scrollHeight);
        }
      });
    };

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

    // Gets the build
    builds.get(fullName, number).then(function (payload) {
      $scope.build = payload.data;
      $scope.task = payload.data.jobs[step - 1];
      $scope.step = step;

      if (['pending', 'killed'].indexOf($scope.task.status) !== -1) {
        // do nothing
      } else if ($scope.task.status === 'running') {
        // stream the build
        stream();
      } else {

        // fetch the logs for the finished build.
        logs.get(fullName, number, step).then(function (payload) {
          var convert = new Filter({stream: false, newline: false});
          var term = document.getElementById("term")
          term.innerHTML = convert.toHtml(payload.data);
        }).catch(function (err) {
          $scope.error = err;
        });
      }
    }).catch(function (err) {
      $scope.error = err;
    });

    repos.subscribe(fullName, function (event) {
      if (event.number !== parseInt(number)) {
        return; // ignore
      }
      // update the build
      $scope.build = event;
      $scope.task = event.jobs[step - 1];
      $scope.$apply();

      // start streaming the current build
      if ($scope.task.status === 'running') {
        stream();
      } else {
        // resets our streaming state
        streaming = false;
      }
    });

    $scope.restart = function () {
      builds.restart(fullName, number).then(function (payload) {
        $scope.build = payload.data;
        $scope.task = payload.data.jobs[step - 1];
      }).catch(function (err) {
        $scope.error = err;
      });
    };

    $scope.cancel = function () {
      builds.cancel(fullName, number).then(function (payload) {
        $scope.build = payload.data;
        $scope.task = payload.data.builds[step - 1];
      }).catch(function (err) {
        $scope.error = err;
      });
    };

    $scope.tail = function () {
      tail = !tail;
    };
  }

  angular
    .module('drone')
    .controller('BuildOutCtrl', BuildOutCtrl)
    .controller('BuildCtrl', BuildCtrl)
    .controller('BuildsCtrl', BuildsCtrl);
})();
