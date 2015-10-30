$(function () {

	// fetch the CSRF token from the meta tag
	var token = $("meta[name='_csrf']").attr("content");

	// ensure every Ajax request has the CSRF token
	// included in the request's header.
	$(document).ajaxSend(function(e, xhr, options) {
		xhr.setRequestHeader("X-CSRF-TOKEN", token);
	});
});
