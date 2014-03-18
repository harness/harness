$(function() {
  var projectForm = $(".form-repo")
  var repoOwnerField = projectForm.find("select[name=owner]")
  var repoNameField = projectForm.find("select[name=name]")

  if(projectForm.length == 0) {
    return
  }

  repoNameField.selectize({
    valueField: 'name',
    labelField: 'name',
    searchField: ['name'],
    create: true
  })

  repoOwnerField.selectize({
    valueField: 'name',
    labelField: 'name',
    searchField: ['name'],
    create: true,
    preload: true,
    load: function(query, callback) {
      var control = repoOwnerField[0].selectize
      control.disable()
      repoNameField[0].selectize.disable()

      $.ajax({
        url: '/new/github.com/available_orgs',
        type: 'GET',
        error: function() {
          callback();
        },
        success: function(orgs) {
          orgs = $.map(orgs, function(o) {
            return { name: o }
          })

          control.enable()
          callback(orgs)
        }
      })
    }
  })

  repoOwnerField.on('change', function() {
    control = repoNameField[0].selectize
    control.disable()
    control.clearOptions()
    orgname = repoOwnerField.val()

    if(orgname == "") return

    $.get('/new/github.com/available_repos',
      { org: orgname },
      function(repos) {
        control.enable()

        $.each(repos, function(i, r) {
          control.addOption({
            name: r.name
          });
        })

        if(repos.length > 0) {
          control.open()
        }
      }
    )
  })

  projectForm.on('submit', function() {
    $("#successAlert").hide();
    $("#failureAlert").hide();
    $('#submitButton').button('loading');

    $.ajax({
      type: 'POST',
      url: projectForm.attr("target"),
      data: projectForm.serialize(),
      success: function(response, status) {
        var name = repoNameField.val()
        var owner = repoOwnerField.val()
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
