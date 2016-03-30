


function RepoViewModel(repo) {
	var self = this;

	Subscribe(repo, function(data){
		var el = $("[data-build="+data.number+"]");

		if (el && el.length !== 0) {
			// find the status label and adjust the
			// build status accordingly.
			var status = el.find(".status");
			status.attr("class", "status "+data.status);
			status.text(data.status);
			return
		}

		// construct the build entry if it doesn't already exist
		// so that we can append to the DOM. The code may not be
		// pretty, but it is simple enough and it works.
		var authoredOrDeployed = "authored"
		var branchOrDeploy = data.branch
		if ( data.event == "deployment" ) {
			authoredOrDeployed = "deployed"
			branchOrDeploy = data.deploy_to
		}

		el = $("<a>").attr("class", "card").attr("href", "/"+repo+"/"+data.number).attr("data-build", data.number)
				.append(
					$("<div>").attr("class", "card-header").append(
						$("<img>").attr("src", data.author_avatar)
					)
				)
				.append(
					$("<div>").attr("class", "card-block").append(
						$("<div>").append(
							$("<div>").attr("class", "status "+ data.status).text(data.status)
						).append(
							$("<h3>").text(data.message)
						)
					).append(
						$("<p>").attr("class","card-text").append(
							$("<em>").text(data.author)
						).append(
							$("<span>").text(authoredOrDeployed)
						).append(
							$("<em>").attr("data-livestamp", data.created_at)
						).append(
							$("<span>").text("to")
						).append(
							$("<em>").text(branchOrDeploy)
						)
					)
				).css("display", "flex").hide().fadeIn(1000);

		// TODO it is very possible that the group may not
		// exist, in which case the we'll need to create the
		// gropu as well.

		// append to the latest group in the timeline.
		$(".card").first().before(el);
	});
}

function Subscribe(repo, _callback) {
	var callback = _callback;
			
	var events = new EventSource("/api/stream/" + repo, {withCredentials: true});
	events.onmessage = function (event) {
		if (callback !== undefined) {
			callback(JSON.parse(event.data));
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



function RepoConfigViewModel(repo) {
	var self = this;

	var timeoutLabel = $(".timeout-label")

	$("input[type='range']").change(function(e) {
		var timeout =  parseInt(e.target.value);
		timeoutLabel.text(timeout + " minutes");
		patchRepo(repo, { timeout: timeout })
	})

	$("#push").change(function(e) {
		patchRepo(repo, {
			allow_push: e.target.checked,
		})
	})

	$("#pull").change(function(e) {
		patchRepo(repo, {
			allow_pr: e.target.checked,
		})
	})

	$("#tag").change(function(e) {
		patchRepo(repo, {
			allow_tag: e.target.checked,
		})
	})

	$("#deploy").change(function(e) {
		patchRepo(repo, {
			allow_deploy: e.target.checked,
		})
	})

	$("#trusted").change(function(e) {
		patchRepo(repo, {
			trusted:  e.target.checked,
		})
	})

	$(".btn-danger").click(function(e) {
		var r = confirm("Are you sure you want to delete this repository?");
		if (r !== false) {
			deleteRepo(repo);
		}
	})
}

function deleteRepo(repo) {
	$.ajax({
		url: "/api/repos/"+repo,
		type: "DELETE",
		contentType: "application/json",
		success: function() {
			window.location.href="/";
		},
	});	
}

function patchRepo(repo, data) {
	$.ajax({
		url: "/api/repos/"+repo,
		type: "PATCH",
		contentType: "application/json",
		data: JSON.stringify(data)
	});	
}