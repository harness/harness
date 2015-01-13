'use strict';

(function () {

  /**
   * fromNow is a helper function that returns a human readable
   * string for the elapsed time between the given unix date and the
   * current time (ex. 10 minutes ago).
   */
  function fromNow() {
    return function(date) {
      return moment(new Date(date*1000)).fromNow();
    }
  }

  /**
   * toDuration is a helper function that returns a human readable
   * string for the given duration in seconds (ex. 1 hour and 20 minutes).
   */
  function toDuration() {
    return function(seconds) {
      return moment.duration(seconds, "seconds").humanize();
    }
  }

  /**
   * toDate is a helper function that returns a human readable
   * string gor the given unix date.
   */
  function toDate() {
    return function(date) {
      return moment(new Date(date*1000)).format('ll');
    }
  }

  /**
   * shortHash is a helper function that returns the shortened
   * version of a commit sha, similar to the --short flag.
   */
  function shortHash() {
    return function(sha) {
      if (sha === undefined) { return ""; }
      return sha.substr(0,10);
    }
  }

  /**
   * gravatar is a helper function that return the user's gravatar
   * image URL given an email hash.
   */
  function gravatar() {
    return function(hash) {
      if (hash  === undefined) { return ""; }
      return "https://secure.gravatar.com/avatar/"+hash+"?s=48&d=mm";
    }
  }

  /**
   * gravatarLarge is a helper function that return the user's gravatar
   * image URL given an email hash.
   */
  function gravatarLarge() {
    return function(hash) {
      if (hash === undefined) { return ""; }
      return "https://secure.gravatar.com/avatar/"+hash+"?s=128&d=mm";
    }
  }

  /**
   * fullName is a helper funcation that returns the full name (slug)
   * for the given repository (ie drone/drone)
   */
  function fullName() {
    return function(repo) {
      if (repo === undefined) { return ""; }
      return repo.owner+"/"+repo.name;
    }
  }

  /**
   * fullName is a helper funcation that returns the full, canonical
   * path for the given repository (ie drone/drone)
   */
  function fullPath() {
    return function(repo) {
      if (repo === undefined) { return ""; }
      return repo.host+"/"+repo.owner+"/"+repo.name;
    }
  }

  /**
   * badgeMarkdown is a helper funcation that returns a markdown string
   * for the given repository's build status badge.
   */
  function badgeMarkdown() {
    return function(repo) {
      if (repo === undefined) { return ""; }
      var scheme = window.location.protocol;
      var host = window.location.host;
      var path = repo.host+'/'+repo.owner+'/'+repo.name;
      var branch = 'master';
      if (repo.scm == 'mercurial') { branch = 'default'; }
      return '[![Build Status]('+scheme+'//'+host+'/api/badge/'+path+'/status.svg?branch='+branch+')]('+scheme+'//'+host+'/'+path+')'
    }
  }

  /**
   * badgeMarkup is a helper funcation that returns an html string
   * for the given repository's build status badge.
   */
  function badgeMarkup() {
    return function(repo) {
      if (repo === undefined) { return ""; }
      var scheme = window.location.protocol;
      var host = window.location.host;
      var path = repo.host+'/'+repo.owner+'/'+repo.name;
      var branch = 'master';
      if (repo.scm == 'mercurial') { branch = 'default'; }
      return '[![Build Status]('+scheme+'//'+host+'/api/badge/'+path+'/status.svg?branch='+branch+')]('+scheme+'//'+host+'/'+path+')'
    }
  }

  /**
   * pullRequests is a helper funcation that filters a list of commits
   * and returns the subset of those commits that are pull requests.
   */
  function pullRequests() {
    return function(commits) {
      var filtered = [];
      angular.forEach(commits, function(commit) {
        if(commit.pull_request.length != 0) {
          filtered.push(commit);
        }
      });
      return filtered;
    }
  }

  /**
   * remoteName is a helper funcation that returns a user-friendly
   * name for the given remote type.
   */
  function remoteName() {
    return function(name) {
      switch (name) {
      case 'gitlab.com'            : return 'GitLab';
      case 'github.com'            : return 'GitHub';
      case 'enterprise.github.com' : return 'GitHub Enterprise';
      case 'bitbucket.org'         : return 'Bitbucket';
      case 'stash.atlassian.com'   : return 'Atlassian Stash';
      case 'gogs'                  : return 'Gogs';
      }
    }
  }

  /**
   * remoteIcon is a helper funcation that returns an icon to represent
   * the given remote type.
   */
  function remoteIcon() {
    return function(name) {
      switch (name) {
      case 'gitlab.com'            : return 'fa-git-square';
      case 'github.com'            : return 'fa-github-square';
      case 'enterprise.github.com' : return 'fa-github-square';
      case 'bitbucket.org'         : return 'fa-bitbucket-square';
      case 'stash.atlassian.com'   : return 'fa-bitbucket-square';
      case 'gogs'                  : return 'fa-git-square';
      }
    }
  }

  /**
   * unique is a helper funcation that returns a unique array
   * of fields from the given list of complex data strucrures.
   * I copied it from Stackoverflow, so don't ask me how it works...
   */
  function unique() {
    return function(input, key) {
        var unique = {};
        var uniqueList = [];
        if (input == undefined) {
          return uniqueList;
        }
        for(var i = 0; i < input.length; i++){
          if(typeof unique[input[i][key]] == "undefined"){
            unique[input[i][key]] = "";
            uniqueList.push(input[i]);
          }
        }
        return uniqueList;
    };
  }

  angular
    .module('app')
    .filter('badgeMarkdown', badgeMarkdown)
    .filter('badgeMarkup', badgeMarkup)
    .filter('fromNow', fromNow)
    .filter('fullName', fullName)
    .filter('fullPath', fullPath)
    .filter('gravatar', gravatar)
    .filter('gravatarLarge', gravatarLarge)
    .filter('pullRequests', pullRequests)
    .filter('remoteIcon', remoteIcon)
    .filter('remoteName', remoteName)
    .filter('shortHash', shortHash)
    .filter('toDate', toDate)
    .filter('toDuration', toDuration)
    .filter('unique', unique);

})();
