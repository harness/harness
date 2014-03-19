;(function ($) {

  "use strict";

  var AddGithubRepo;

  AddGithubRepo = {
    formSelector:       ".form-repo",
    ownerFieldSelector: "select[name=owner]",
    nameFieldSelector:  "select[name=name]",
    orgsUrl:            "/new/github.com/available_orgs",
    reposUrl:           "/new/github.com/available_repos",

    initialize: function() {
      this.form = $(this.formSelector);

      if (this.form.length === 0) {
        return;
      }

      this.nameField  = this.form.find(this.nameFieldSelector);
      this.ownerField = this.form.find(this.ownerFieldSelector);

      this.initValidation();
      this.initSelectize();
      this.initSwitcher();
    },

    initSelectize: function() {
      var that = this;

      this.nameField.selectize({
        valueField:  "name",
        labelField:  "name",
        searchField: [ "name" ],
        create:      true
      });

      this.ownerField.selectize({
        valueField:  "name",
        labelField:  "name",
        searchField: [ "name" ],
        create:      true,
        preload:     true,
        load: function(query, callback) {
          var control;

          control = that.ownerField[0].selectize;
          control.disable();
          that.nameField[0].selectize.disable();

          $.get(that.orgsUrl, function (orgs) {
            control.enable();

            callback($.map(orgs, function (org) {
              return { name: org };
            }));
          }).fail(function () {
            callback({ });
          });
        }
      });
    },

    initValidation: function() {
      var that = this;

      this.form.on("submit", function() {
        $("#successAlert").hide();
        $("#failureAlert").hide();
        $("#submitButton").button("loading");

        $.ajax({
          type: "POST",
          url:  that.form.attr("target"),
          data: that.form.serialize(),
          success: function(response, status) {
            var name, owner, domain;

            name   = that.nameField.val();
            owner  = that.ownerField.val();
            domain = $("input[name=domain]").val();

            window.location.pathname = "/" + domain + "/" + owner + "/" + name;
          },
          error: function() {
            $("#failureAlert").text("Unable to setup the Repository");
            $("#failureAlert").show().removeClass("hide");
            $("#submitButton").button("reset");
          }
        });

        return false;
      });
    },

    initSwitcher: function() {
      var that = this;

      this.ownerField.on("change", function() {
        var control, orgname;

        control = that.nameField[0].selectize;
        control.disable();
        control.clearOptions();

        orgname = that.ownerField.val();

        if (orgname === "") {
          return;
        }

        $.get(that.reposUrl, { org: orgname }, function (repos) {
          control.enable();

          $.each(repos, function (i, repo) {
            control.addOption({ name: repo.name });
          });

          if (repos.length > 0) {
            control.open();
          }
        });
      });
    }
  };

  // Init on DOM ready

  $(function () {
    AddGithubRepo.initialize();
  });

})(jQuery);
