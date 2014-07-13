'use strict';

angular.module('app').filter('gravatar', function() {
  return function(gravatar) {
    return "https://secure.gravatar.com/avatar/"+gravatar+"?s=48&d=mm"
  }
});

angular.module('app').filter('fromNow', function() {
  return function(date) {
    return moment(new Date(date*1000)).fromNow();
  }
});

angular.module('app').filter('toDuration', function() {
  return function(seconds) {
  	return moment.duration(seconds, "seconds").humanize();
  }
});

angular.module('app').filter('toDate', function() {
  return function(date) {
    return moment(new Date(date*1000)).format('ll');
  }
});

angular.module('app').filter('fullName', function() {
  return function(repo) {
    return repo.owner+"/"+repo.name;
  }
});

angular.module('app').filter('fullPath', function() {
  return function(repo) {
    if (repo == undefined) { return ""; }
    return repo.remote+"/"+repo.owner+"/"+repo.name;
  }
});

angular.module('app').filter('shortHash', function() {
  return function(sha) {
  	if (!sha) { return ""; }
    return sha.substr(0,10)
  }
});

angular.module('app').filter('badgeMarkdown', function() {
  return function(repo) {
    if (repo == undefined) { return ""; }
    var scheme = window.location.protocol;
    var host = window.location.host;
    var path = repo.host+'/'+repo.owner+'/'+repo.name;
    return '[![Build Status]('+scheme+'//'+host+'/v1/badge/'+path+'/status.svg?branch=master)]('+scheme+'//'+host+'/'+path+')'
  }
});

angular.module('app').filter('badgeMarkup', function() {
  return function(repo) {
    if (repo == undefined) { return ""; }
    var scheme = window.location.protocol;
    var host = window.location.host;
    var path = repo.host+'/'+repo.owner+'/'+repo.name;
    return '<a href="'+scheme+'//'+host+'/'+path+'"><img src="'+scheme+'//'+host+'/v1/badge/'+path+'/status.svg?branch=master" /></a>'
  }
});

angular.module('app').filter('remoteName', function() {
  return function(name) {
    switch (name) {
    case 'gitlab.com'            : return 'GitLab';
    case 'github.com'            : return 'GitHub';
    case 'enterprise.github.com' : return 'GitHub Enterprise';
    case 'bitbucket.org'         : return 'Bitbucket';
    case 'stash.atlassian.com'   : return 'Atlassian Stash';
    }
  }
});

angular.module('app').filter('remoteIcon', function() {
  return function(name) {
    switch (name) {
    case 'gitlab.com'            : return 'fa-git-square';
    case 'github.com'            : return 'fa-github-square';
    case 'enterprise.github.com' : return 'fa-github-square';
    case 'bitbucket.org'         : return 'fa-bitbucket-square';
    case 'stash.atlassian.com'   : return 'fa-bitbucket-square';
    }
  }
});


angular.module('app').filter('unique', function() {
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
});