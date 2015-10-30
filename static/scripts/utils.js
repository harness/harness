Array.prototype.remove = function() {
	var what, a = arguments, L = a.length, ax;
	while (L && this.length) {
		what = a[--L];
		while ((ax = this.indexOf(what)) !== -1) {
			this.splice(ax, 1);
		}
	}
	return this;
};

function escapeHTML(html) {
    return document.createElement('div').appendChild(
    	document.createTextNode(html)).parentNode.innerHTML;
}
