'use strict';

(function () {

	/**
	 * author is a helper function that return the builds
	 * commit or pull request author.
	*/
	function author() {
		return function(build) {
			if (!build) { return ""; }
			if (!build.head_commit && !build.pull_request) { return ""; }
			if (build.head_commit) { return build.head_commit.author.login || ""; }
			return build.pull_request.source.author.login;
		}
	}

	/**
	 * sha is a helper function that return the builds sha.
	*/
	function sha() {
		return function(build) {
			if (!build) { return ""; }
			if (!build.head_commit && !build.pull_request) { return ""; }
			if (build.head_commit) { return build.head_commit.sha || ""; }
			return build.pull_request.source.sha;
		}
	}

	/**
	 * ref is a helper function that return the builds sha.
	*/
	function ref() {
		return function(build) {
			if (!build) { return ""; }
			if (!build.head_commit && !build.pull_request) { return ""; }
			if (build.head_commit) { return build.head_commit.ref || ""; }
			return build.pull_request.source.ref;
		}
	}

	/**
	 * message is a helper function that return the builds message.
	*/
	function message() {
		return function(build) {
			if (!build) { return ""; }
			if (!build.head_commit && !build.pull_request) { return ""; }
			if (build.head_commit) { return build.head_commit.message || ""; }
			return build.pull_request.title || "";
		}
	}

	angular
		.module('drone')
		.filter('author', author)
		.filter('message', message)
		.filter('sha', sha)
		.filter('ref', ref);

})();
