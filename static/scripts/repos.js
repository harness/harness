




function RepoListViewModel(repos) {
	var self = this;

	var mapped = $.map(repos, function(repo) {
		return new Repo(repo)
	});

	self.repos = ko.observableArray(mapped);
	self.newRepo = ko.observable();

	self.addRepo = function() {
		$.ajax({
			url: "/api/repos/"+self.newRepo(),
			type: "POST",
			contentType: "application/json",
			success: function( data ) {
				self.repos.push(new Repo(data));
				self.repos.sort(RepoCompare);
				self.newRepo("");
			},
			error: function( data ) {
				console.log(data);
			}
		});
	};
}
