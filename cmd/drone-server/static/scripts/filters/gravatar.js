'use strict';

(function () {

	/**
	 * gravatar is a helper function that return the user's gravatar
	 * image URL given an email hash.
	*/
	function gravatar() {
		return function(hash) {
			if (!hash) { return "http://www.gravatar.com/avatar/00000000000000000000000000000000?d=mm&f=y"; }
			return "https://secure.gravatar.com/avatar/"+hash+"?s=48&d=mm";
		}
	}

	/**
	 * gravatarLarge is a helper function that return the user's gravatar
	 * image URL given an email hash.
	 */
	function gravatarLarge() {
		return function(hash) {
			if (!hash) { return "http://www.gravatar.com/avatar/00000000000000000000000000000000?d=mm&f=y"; }
			return "https://secure.gravatar.com/avatar/"+hash+"?s=128&d=mm";
		}
	}

	angular
		.module('drone')
		.filter('gravatar', gravatar)
		.filter('gravatarLarge', gravatarLarge)

})();
