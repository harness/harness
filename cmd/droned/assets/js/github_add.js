$(function() {
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

  var projectForm = $(document.forms[0])

  repoOwnerField = $("input[name=owner]")
  repoNameField = $("input[name=name]")

  projectsList.on('click', 'a', function(event) {
    var badge = $(event.target).parent()

    repoOwnerField.val(badge.data('owner'))
    repoNameField.val(badge.data('name'))
    repoNameField.focus()

    $(document).scrollTop(projectForm.offset().top)

    return false
  })

  projectForm.on('submit', function(event) {
    $("#successAlert").hide();
    $("#failureAlert").hide();
    $('#submitButton').button('loading');

    var form = $(event.target)

    $.ajax({
      type: "POST",
      url: form.attr("target"),
      data: form.serialize(),
      success: function(response, status) {
        console.log(response, status)
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
