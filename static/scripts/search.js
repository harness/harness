var repoExpr = /.+\/.+/;

var remoteRepos = new Bloodhound({
    queryTokenizer: Bloodhound.tokenizers.whitespace,
    datumTokenizer: Bloodhound.tokenizers.obj.whitespace("full_name"),

	identify: function(obj) { return obj.full_name; },
	prefetch: '/api/user/repos/remote'
});


function reposWithDefaults(q, sync) {
  if (q === "") {
  	sync(remoteRepos.all())
  } else {
    remoteRepos.search(q, sync);
  }
}

$('.typeahead').typeahead({
	hint: true,
	highlight: true,
	minLength: 0
},
{
	name: "repos",
	display: "full_name",
	source: reposWithDefaults,
	templates: {
		empty: function(obj) {
			if (obj.query.match(repoExpr) !== null) {
				return [
					"<div>",
						"<div class='not-indexed-message'>",
							"<p>",
								"No matches found for",
								"<em>",
								obj.query,
								"</em>",
							"</p>",
							"<p>",
								"This repository may not be indexed yet.",
								"<a href='/"+obj.query+"'>",
								"Click here",
								"</a>",
								"to visit this repository page directly.",
							"</p>",
						"</div>",
					"</div>"
				].join("\n");
			}
			return [
				"<div>",
					"<div class='no-matches-message'>",
						"No matches found",
					"</div>",
				"</div>"
			].join("\n");
		},
		suggestion: function(obj) {
			return "<div><div><img src='"+obj.avatar_url+"' width='32px' height='32px' /></div><div>"+ obj.full_name +"</div></div>";
		}
	}
}).bind('typeahead:select', function(ev, suggestion) {
	window.location.href="/"+suggestion.full_name;
});
