/**
 * Creates an observable build job.
 */
function Job(data) {
	this.number	  = ko.observable(data.number);
	this.commit	  = ko.observable(data.commit);
	this.started_at  = ko.observable(data.started_at);
	this.finished_at = ko.observable(data.finished_at);
	this.exit_code   = ko.observable(data.exit_code);
	this.status	  = ko.observable(data.status);
	this.environment = ko.observable(data.environment);
}

/**
 * Creates an observable build.
 */
function Build(data) {
	this.number      = ko.observable(data.number);
	this.commit      = ko.observable(data.commit);
	this.branch      = ko.observable(data.branch);
	this.author      = ko.observable(data.author);
	this.message     = ko.observable(data.message);
	this.status      = ko.observable(data.status);
	this.event       = ko.observable(data.event);
	this.started_at  = ko.observable(data.started_at);
	this.finished_at = ko.observable(data.finished_at);
}

/**
 * Creates an observable repository.
 */
function Repo(data) {
	this.full_name  = ko.observable(data.full_name);
	this.owner      = ko.observable(data.owner);
	this.name       = ko.observable(data.name);
	this.private    = ko.observable(data.private);
	this.trusted    = ko.observable(data.trusted);
	this.timeout    = ko.observable(data.timeout);
	this.avatar_url = ko.observable(data.avatar_url);
	this.clone_url  = ko.observable(data.clone_url);
	this.link_url   = ko.observable(data.link_url);
	this.starred	= ko.observable(data.starred || false);
	this.events     = ko.observable(data.events);
	this.hook       = ko.observable(new Hook(data));
}

/**
 * Compares two repository objects by name. Used to sort
 * a list of repositories.
 */
function RepoCompare(a, b) {
	return a.full_name().toLowerCase() > b.full_name().toLowerCase() ? 1 : -1; 
}

/**
 * Creates an observable object that stores a list of hook event
 * types (push, pull request, etc) and true or false if enabled.
 */
function Hook(repo) {
	var data = {
		"pull_request" : repo.events.indexOf("pull_request") !== -1,
		"push"         : repo.events.indexOf("push")         !== -1,
		"tag"          : repo.events.indexOf("tag")          !== -1,
		"deploy"       : repo.events.indexOf("deploy")       !== -1
	};

	this.pull_request = ko.observable(data.pull_request);
	this.push         = ko.observable(data.push);
	this.tag          = ko.observable(data.tag);
	this.deploy       = ko.observable(data.deploy);
}

/**
 * Creates an observable user.
 */
function User(data) {
	this.login = ko.observable(data.login);
	this.email = ko.observable(data.email);
	this.avatar_url = ko.observable(data.avatar_url);
	this.active = ko.observable(data.active);
	this.admin = ko.observable(data.admin);
}

/**
 * Compares two user objects by login. Used to sort
 * a list of users.
 */
function UserCompare(a, b) {
	return a.login().toLowerCase() > b.login().toLowerCase() ? 1 : -1;  
}
