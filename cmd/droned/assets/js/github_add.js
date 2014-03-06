$(function() {
  var projectForm = $(".form-repo")
  var repoOwnerField = projectForm.find("input[name=owner]")
  var repoNameField = projectForm.find("input[name=name]")

  var projectsBlock = $(".github-projects")
  var projectsList = projectsBlock.find(".projects-list")
  var spinnerBlock = projectsBlock.find(".spinner")

  spinner = new Spinner({
    lines: 12,
    speed: 0.5,
    width: 4,
    length: 10
  })
  spinner.spin(spinnerBlock[0])

  GithubRepos.get(function(response) {
    $.each(response, function(i, repo) {
      var title = repo.owner + "/" + repo.name

      item = $("<div></div>").addClass("item")
      link = $("<a></a>").text(title).attr("href", "#")
      if(repo.private) {
        icon = $("<span></span>").addClass("glyphicon").addClass("glyphicon-lock")
        item.append("&nbsp;")
        item.append(icon)
      }

      item.append(link)

      item.data('owner', repo.owner)
      item.data('name', repo.name)

      projectsList.append(item)
    })

    spinner.stop()
    spinnerBlock.remove()
  })

  projectsList.on('click', 'a', function(event) {
    var badge = $(event.target).parent()

    repoOwnerField.val(badge.data('owner'))
    repoNameField.val(badge.data('name'))
    repoNameField.focus()

    $(document).scrollTop(projectForm.offset().top)

    return false
  })

  projectForm.on('submit', function() {
    $("#successAlert").hide();
    $("#failureAlert").hide();
    $('#submitButton').button('loading');

    $.ajax({
      type: "POST",
      url: projectForm.attr("target"),
      data: projectForm.serialize(),
      success: function(response, status) {
        var name = $("input[name=name]").val()
        var owner = $("input[name=owner]").val()
        var domain = $("input[name=domain]").val()
        window.location.pathname = "/" + domain + "/"+owner+"/"+name
      },
      error: function() {
        $("#failureAlert").text("Unable to setup the Repository");
        $("#failureAlert").show().removeClass("hide");
        $('#submitButton').button('reset');
      }
    });

    return false;
  })
})
