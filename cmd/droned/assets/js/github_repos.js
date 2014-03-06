GithubRepos = {
  url: "/new/github.com/available_repos",
  get: function(success) {
    $.getJSON(this.url, function(response) {
      success(response)
    })
  }
}
