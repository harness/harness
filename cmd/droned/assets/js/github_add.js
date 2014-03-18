AddGithubRepo = {
  formSelector: ".form-repo",
  ownerFieldSelector: "select[name=owner]",
  nameFieldSelector: "select[name=name]",
  orgsUrl: '/new/github.com/available_orgs',
  reposUrl: '/new/github.com/available_repos',

  selectize: function() {
    var that = this

    this.nameField.selectize({
      valueField: 'name',
      labelField: 'name',
      searchField: ['name'],
      create: true
    })

    this.ownerField.selectize({
      valueField: 'name',
      labelField: 'name',
      searchField: ['name'],
      create: true,
      preload: true,
      load: function(query, callback) {
        var control = that.ownerField[0].selectize
        control.disable()
        that.nameField[0].selectize.disable()

        $.ajax({
          url: that.orgsUrl,
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

    this.bindSwitcher()
  },

  bindSwitcher: function() {
    var that = this
    this.ownerField.on('change', function() {
      control = that.nameField[0].selectize
      control.disable()
      control.clearOptions()
      orgname = that.ownerField.val()

      if(orgname == "") return

      $.get(that.reposUrl,
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
  },

  validation: function() {
    var that = this
    this.form.on('submit', function() {
      $("#successAlert").hide();
      $("#failureAlert").hide();
      $('#submitButton').button('loading');

      $.ajax({
        type: 'POST',
        url: that.form.attr("target"),
        data: that.form.serialize(),
        success: function(response, status) {
          var name = that.nameField.val()
          var owner = that.ownerField.val()
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
  },
  start: function() {
    this.form = $(this.formSelector)
    if(this.form.length == 0) {
      return
    }

    this.nameField = this.form.find(this.nameFieldSelector)
    this.ownerField = this.form.find(this.ownerFieldSelector)

    this.validation()
    this.selectize()
  }
}

$(function() {
  AddGithubRepo.start()
})
