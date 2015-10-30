
function NodeViewModel() {
	var self = this;

	// handle requests to create a new node.
	$(".modal-node button").click(function(e) {

		var node = {
			address : $("#addr").val(),
			key     : $("#key").val(),
			cert    : $("#cert").val(),
			ca      : $("#ca").val()
		};

		$.ajax({
			url: "/api/nodes",
			type: "POST",
			contentType: "application/json",
			data: JSON.stringify(node),
			success: function( data ) {
				// clears the form value
				$(".modal-node input").val("");

				var el = $("<div>").attr("class", "col-sm-4").append(
					$("<div>").attr("class", "card").attr("data-id", data.id).append(
						$("<div>").attr("class", "card-header").append(
							$("<i>").attr("class", "linux_amd64")
						)
					).append(
						$("<div>").attr("class", "card-block").append(
							$("<h3>").text(data.address)
						).append(
							$("<p>").attr("class", "card-text").text(data.architecture)
						).append(
							$("<div>").attr("class", "btn-group").append(
								$("<button>").attr("class","btn btn-danger").text("Delete")
							)
						)
					)
				)

				$( ".node-row" ).prepend(el);
			},
			error: function( data ) {
 				console.log(data);
			}
		});
	});


	$(".node-row").on('click', '.btn-group .btn-danger', function(){
		// gets the unique identifier for the click row, which is
		// the user login id.
		var id = $( this ).context
					.parentNode
					.parentNode
					.parentNode.dataset.id;
		
		var r = confirm("Are you sure you want to delete node "+id+"?");
		if (r === false) {
			return;
		}

		// makes an API call to delete the user and then, if successful,
		// removes from the list.
		$.ajax({
			url: "/api/nodes/"+id,
			type: "DELETE",
			success: function( data ) {
				$("[data-id='"+id+"']").parent().remove();
			}
		});
	});
}
