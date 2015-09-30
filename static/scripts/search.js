

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
		suggestion: function(obj) {
			return "<div><div><img src='"+obj.avatar_url+"' width='32px' height='32px' /></div><div>"+ obj.full_name +"</div></div>";
		}
	}
}).bind('typeahead:select', function(ev, suggestion) {
	window.location.href="/"+suggestion.full_name;
});
