
function UserViewModel() {
	var self = this;

	// handle requests to create a new user.
	$(".modal-user button").click(function(e) {
		var login = $(".modal-user input").val();
		var user = { login: login };

		$(".alert-danger").hide();

		$.ajax({
			url: "/api/users",
			type: "POST",
			data: JSON.stringify(user),
			contentType: "application/json",
			success: function( data ) {
				// clears the form value
				$(".modal-user input").val("");

				// find an existing item in the list and clone it.
				var el = $(".col-sm-4").first().clone();
				el.find("img").attr("src", data.avatar_url);
				el.find("h3").text(data.login);
				el.find("p").text(data.email);
				el.find(".card").attr("data-id", data.login);
				el.find(".card").attr("data-admin", data.admin);

				$( ".user-row" ).prepend(el);
			},
			error: function(data) {
				$(".alert-danger").text(data.responseText);
				$(".alert-danger").show();
			}
		});
	});


	$(".user-row").on('click', '.btn-group .btn-info', function(){ 
		// gets the unique identifier for the click row, which is
		// the user login id.
		var row   = $( this ).closest("[data-id]");
		var id    = row.data("id");
		var admin = row.data("admin") == true;

		row.attr("data-admin", !admin);

		$.ajax({
			url: "/api/users/"+id,
			type: "PATCH",
			data: JSON.stringify({active:true, admin: !admin}),
			contentType: "application/json"
		});

	}); 

	$(".user-row").on('click', '.btn-group .btn-danger', function(){
		// gets the unique identifier for the click row, which is
		// the user login id.
		var row = $( this ).closest("[data-id]");
		var id  = row.data("id");
		
		var r = confirm("Are you sure you want to delete "+id+"?");
		if (r === false) {
			return;
		}

		// makes an API call to delete the user and then, if successful,
		// removes from the list.
		$.ajax({
			url: "/api/users/"+id,
			type: "DELETE",
			success: function( data ) {
				row.parent().remove();
			}
		});
	});
}
