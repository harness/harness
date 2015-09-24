'use strict';

(function () {

  /**
   * The RepoService provides access to repository
   * data using REST API calls.
   */
  function RepoService($http, $window) {

    var callback,
      websocket,
      token = localStorage.getItem('access_token');

    /**
     * Gets a list of all repositories.
     */
    this.list = function () {
      return $http.get('api/user/repos');
    };

    /**
     * Gets a repository by name.
     *
     * @param {string} Name of the repository.
     */
    this.get = function (repoName) {
      return $http.get('api/repos/' + repoName);
    };

    /**
     * Creates a new repository.
     *
     * @param {object} JSON representation of a repository.
     */
    this.post = function (repoName) {
      return $http.post('api/repos/' + repoName);
    };

    /**
     * Updates an existing repository.
     *
     * @param {object} JSON representation of a repository.
     */
    this.update = function (repo) {
      return $http.patch('api/repos/' + repo.full_name, repo);
    };

    /**
     * Deletes a repository.
     *
     * @param {string} Name of the repository.
     */
    this.delete = function (repoName) {
      return $http.delete('api/repos/' + repoName);
    };

    /**
     * Watch a repository.
     *
     * @param {string} Name of the repository.
     */
    this.watch = function (repoName) {
      return $http.post('api/repos/' + repoName + '/watch');
    };

    /**
     * Unwatch a repository.
     *
     * @param {string} Name of the repository.
     */
    this.unwatch = function (repoName) {
      return $http.delete('api/repos/' + repoName + '/unwatch');
    };

    /**
     * Encrypt the set of parameters.
     *
     * @param {string} Name of the repository.
     * @param {string} Plaintext to encrypt.
     */
    this.encrypt = function (repoName, plaintext) {
      var conf = {
        headers: {
          'Content-Type': 'text/plain; charset=UTF-8'
        }
      }

      return $http.post('api/repos/' + repoName + '/encrypt', btoa(plaintext), conf);
    };

    var callback,
      events,
      token = localStorage.getItem('access_token');

    /**
     * Subscribes to a live update feed for a repository
     *
     * @param {string} Name of the repository.
     */
    this.subscribe = function (repo, _callback) {
      callback = _callback;

      events = new EventSource("api/stream/" + repo + "?access_token=" + token, {withCredentials: true});
      events.onmessage = function (event) {
        if (callback !== undefined) {
          callback(angular.fromJson(event.data));
        }
      };
      events.onerror = function (event) {
        callback = undefined;
        if (events !== undefined) {
          events.close();
          events = undefined;
        }
        console.log('user event stream closed due to error.', event);
      };
    };

    this.unsubscribe = function () {
      callback = undefined;
      if (events !== undefined) {
        events.close();
        events = undefined;
      }
    };
  }

  angular
    .module('drone')
    .service('repos', RepoService);
})();
