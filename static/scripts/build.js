

function JobViewModel(repo, build, job, status) {
	var self = this;
	self.status = status;

	self.stream = function() {
		$( "#output" ).html("");
		$("#restart").hide();
		$("#cancel").show();

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
	};

	if (status !== "running" && status !== "pending") {
		Logs(repo, build, job);
		$("#restart").show();
	}

	if (status === "running") {
		self.stream();
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

	$("#cancel").click(function() {
		$("#cancel").hide();

		$.ajax({
			url: "/api/repos/"+repo+"/builds/"+build+"/"+job,
			type: "DELETE",
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
		if (data.number !== build) {
			return;
		}

		var before = self.status;
		self.status = data.jobs[job-1].status;

		// update the status for each job in the view
		for (var i=0;i<data.jobs.length;i++) {
			var job_ = data.jobs[i];
			var el = $("[data-job="+job_.number+"]");

			el.find(".status")
				.attr("class", "status "+job_.status).text(job_.status);

			switch (job_.status) {
			case "running":
				el.find(".msg-running").find("span").attr("data-livestamp", job_.started_at);

				el.find(".msg-pending").hide();
				el.find(".msg-running").show();
				el.find(".msg-finished").hide();
				el.find(".msg-exited").hide();
				break;
			case "pending":
				el.find(".msg-pending").show();
				el.find(".msg-running").hide();
				el.find(".msg-finished").hide();
				el.find(".msg-exited").hide();
				break;
			default:
				el.find(".msg-finished").find("span").attr("data-livestamp", job_.finished_at);
				el.find(".msg-exited").find("span").text(job_.exit_code);

				el.find(".msg-pending").hide();
				el.find(".msg-running").hide();
				el.find(".msg-finished").show();
				el.find(".msg-exited").show();
				break;
			}
		}

		var after = self.status;

		// if the status has changed we should start
		// streaming the build contents.
		if (before !== after && after === "running") {
			self.stream();
		}

		// if the status is changes to complete, we can show
		// the restart button and hide the tail button.
		if (after !== "pending" && after !== "running") {
			$("#restart").show();
			$("#cancel").hide();
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
