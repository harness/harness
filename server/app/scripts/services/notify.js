'use strict';

angular.module('app').service('notify', ['$window', '$timeout', '$location', function($window, $timeout, $location) {

	// returns true if the HTML5 Notifications API is supported.
	this.supported = function() {
		return ("Notification" in $window)
	}

	// returns true if the user has granted permission to 
	// display HTML5 notifications.
	this.granted = function() {
		return ("Notification" in $window) && Notification.permission === "granted";
	}

	// instructs the browser to request permission to
	// display HTML5 notifications.
	this.requestPermission = function() {
		Notification.requestPermission();
	}

	// sends an HTML5 desktop notification using the specified
	// title and notification options (optional).
	this.send = function(title, opts) {
		if ("Notification" in $window) {
			var notification = new Notification(title, opts);

			// automatically close the notification after
			// 10 seconds of being open.
			$timeout(function() {
				notification.close();
			}, 5000);

			// if a hyperlink was specified, open the link
			// when the notification is clicked.
			notification.onclick = function() {
				if (opts.href == undefined) {
					return;
				}
				// not exactly sure why this is even necessary:
				// http://stackoverflow.com/questions/11784656/angularjs-location-not-changing-the-path
				$timeout(function(){ 
					$location.path(opts.href);
				}, 1);
			};
		}
	};

	// sends an HTML5 desktop notification for a Commit.
	this.sendCommit = function(repo, commit) {
		// ignore 'Pending' messages since they are (usually) immediately
		// followed-up by a 'Started' message, and we don't want to flood
		// the user with notifications.
		if (commit.status == 'Pending') {
			return;
		}

		var title = repo.owner+'/'+repo.name;
		var url = '/'+repo.host+'/'+repo.owner+'/'+repo.name+'/'+commit.branch+'/'+commit.sha;

		this.send(title, {
			icon: 'https://secure.gravatar.com/avatar/'+commit.gravatar,
			body: commit.message,
			href: url,
		});
	}
}]);
