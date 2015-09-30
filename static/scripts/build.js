

function JobViewModel(repo, build, job, status) {
	var self = this;
	self.status = status;


	if (status !== "running" && status !== "pending") {
		Logs(repo, build, job);
	}

	if (status === "running") {
		Stream(repo, build, job, function(out){
			$( "#output" ).append(out);
		});
	}

	$("#restart").click(function() {
		$("#restart").hide();
		$("#output").html("");
		$(".status").attr("class", "status pending").text("pending");

		$.ajax({
			url: "/api/repos/"+repo+"/builds/"+build,
			type: "POST",
			success: function( data ) { },
			error: function( data ) {
				console.log(data);
			}
		});
	})

			
	Subscribe(repo, function(data){
		if (!data.jobs) {
			return;
		}

		var before = self.status;
		self.status = data.jobs[job-1].status;

		// update the status for each job in the view
		for (var i=0;i<data.jobs.length;i++) {
			var job_ = data.jobs[i];
			$("[data-job="+job_.number+"]").find(".status")
				.attr("class", "status "+job_.status).text(job_.status);
		}

		var after = self.status;

		// if the status has changed we should start
		// streaming the build contents.
		if (before !== after && after === "running") {
			$( "#output" ).html("");

			var buf = new Drone.Buffer();
			buf.start(document.getElementById("output"));

			$( "#tail" ).show();
			$( "#tail" ).click(function() {
				buf.autoFollow = !buf.autoFollow;
				if (buf.autoFollow) {
					$( "#tail i" ).text("pause");
					$( "#tail" ).show();

					// scroll to the bottom of the page
					window.scrollTo(0, document.body.scrollHeight);
				} else {
					$( "#tail i" ).text("expand_more");
					$( "#tail" ).show();
				}
				
			})

			Stream(repo, build, job, function(out){
				buf.write(out);
			});
		}

		// if the status is changes to complete, we can show
		// the restart button and hide the tail button.
		if (after !== "pending" && after !== "running") {
			$("#restart").show();
			$("#tail").hide();
		}
	}.bind(this));
}




function Logs(repo, build, job) {

	$.get( "/api/repos/"+repo+"/logs/"+build+"/"+job, function( data ) {

		var convert = new Filter({stream: false, newline: false});
		var escaped = convert.toHtml(escapeHTML(data));

		$( "#output" ).html( escaped );
	});
}

function Stream(repo, build, job, _callback) {
	var callback = _callback;

	var events = new EventSource("/api/stream/" + repo + "/" + build + "/" + job, {withCredentials: true});
	events.onmessage = function (event) {
		if (callback !== undefined) {
			callback(event.data);
		}
	};
	events.onerror = function (event) {
		callback = undefined;
		if (events !== undefined) {
			events.close();
			events = undefined;
		}
		console.log('user event stream closed due to error.', event);
	};
};
