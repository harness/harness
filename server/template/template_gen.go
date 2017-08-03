package template

import "html/template"

// list of embedded template files.
var files = []struct {
	name string
	data string
}{
	{
		name: "error.html",
		data: error,
	}, {
		name: "index.html",
		data: index,
	}, {
		name: "index_polymer.html",
		data: indexpolymer,
	}, {
		name: "login.html",
		data: login,
	}, {
		name: "logout.html",
		data: logout,
	}, {
		name: "script.html",
		data: script,
	},
}

// T exposes the embedded templates.
var T *template.Template

func init() {
	T = template.New("_").Funcs(funcMap)
	for _, file := range files {
		T = template.Must(
			T.New(file.name).Parse(file.data),
		)
	}
}

//
// embedded template files.
//

// files/error.html
var error = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8"/>
  <meta content="width=device-width, initial-scale=1" name="viewport"/>
  <meta content="ie=edge" http-equiv="x-ua-compatible"/>
  <link href="https://fonts.googleapis.com/css?family=Roboto" rel="stylesheet"/>
  <link href="https://fonts.googleapis.com/css?family=Roboto+Mono" rel="stylesheet"/>
  <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet"/>
  <link href="/static/favicon.ico" rel="icon" type="image/x-icon"/>
  <link rel="stylesheet" href="/static/app.css" />
  <title>error | drone</title>
</head>
<body>
  {{ .error }}
</body>
</html>
`

// files/index.html
var index = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="author" content="bradrydzewski">
	<meta name="viewport" content="width=device-width, minimum-scale=1, initial-scale=1, user-scalable=yes">

	<link rel="shortcut icon" type="image/png" sizes="32x32" href="/favicon-32x32.png">
	<link rel="shortcut icon" type="image/png" sizes="16x16" href="/favicon-16x16.png">

	<title></title>
	<script>
			window.ENV = {};
			window.ENV.server = window.location.protocol+"//"+window.location.host;
			{{ if .csrf }}window.ENV.csrf = "{{ .csrf }}"{{ end }}
			{{ if .user }}
			window.USER = {{ json .user }};
			{{ end }}
	</script>
	<script>
		window.WebComponents = window.WebComponents || {};
		window.WebComponents.root = '//cdnjs.cloudflare.com/ajax/libs/webcomponentsjs/1.0.4/';
		// inline polymer loader
		!function(){"use strict";window.WebComponents=window.WebComponents||{};var e="webcomponents-loader.js",t=[];if("import"in document.createElement("link")||t.push("hi"),(!("attachShadow"in Element.prototype&&"getRootNode"in Element.prototype)||window.ShadyDOM&&window.ShadyDOM.force)&&t.push("sd"),(!window.customElements||window.customElements.forcePolyfill)&&t.push("ce"),"content"in document.createElement("template")&&window.Promise&&Array.from&&document.createDocumentFragment().cloneNode()instanceof DocumentFragment||(t=["lite"]),t.length){var n,o="webcomponents-"+t.join("-")+".js";if(window.WebComponents.root)n=window.WebComponents.root+o;else{var c=document.querySelector('script[src*="'+e+'"]');n=c.src.replace(e,o)}var r=document.createElement("script");r.src=n,"loading"===document.readyState?document.write(r.outerHTML):document.head.appendChild(r)}else{var d=function(){requestAnimationFrame(function(){window.WebComponents.ready=!0,document.dispatchEvent(new CustomEvent("WebComponentsReady",{bubbles:!0}))})};"loading"!==document.readyState?d():document.addEventListener("readystatechange",function a(){d(),document.removeEventListener("readystatechange",a)})}}();
	</script>

	<link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Roboto">
	<link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Roboto+Mono">
	<link rel="import" href="/src/drone/drone-app.html">

	<style>
		html, body {
			padding:0px;
			margin:0px;
		}
	</style>
</head>
<body>
	<drone-app></drone-app>
</body>
</html>
`

// files/index_polymer.html
var indexpolymer = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="author" content="bradrydzewski">
	<meta name="viewport" content="width=device-width, minimum-scale=1, initial-scale=1, user-scalable=yes">

	<link rel="shortcut icon" type="image/png" sizes="32x32" href="/favicon-32x32.png">
	<link rel="shortcut icon" type="image/png" sizes="16x16" href="/favicon-16x16.png">

	<title></title>
	<script>
			window.ENV = {};
			window.ENV.server = window.location.protocol+"//"+window.location.host;
			{{ if .csrf }}window.ENV.csrf = "{{ .csrf }}"{{ end }}
			{{ if .user }}
			window.USER = {{ json .user }};
			{{ end }}
	</script>
	<script>
		// TODO load a polyfill for SSE for the Edge browser. Consider downloading
		// with bower instead of loading from cndjs.
		if (!window.EventSource) {
			var ssePolyfill = document.createElement("script");
			ssePolyfill.src = "https://cdnjs.cloudflare.com/ajax/libs/event-source-polyfill/0.0.9/eventsource.min.js";
			document.body.appendChild(ssePolyfill);
		}
	</script>
	<script>
		window.WebComponents = window.WebComponents || {};
		window.WebComponents.root = '//cdnjs.cloudflare.com/ajax/libs/webcomponentsjs/1.0.4/';
		// inline polymer loader
		!function(){"use strict";window.WebComponents=window.WebComponents||{};var e="webcomponents-loader.js",t=[];if("import"in document.createElement("link")||t.push("hi"),(!("attachShadow"in Element.prototype&&"getRootNode"in Element.prototype)||window.ShadyDOM&&window.ShadyDOM.force)&&t.push("sd"),(!window.customElements||window.customElements.forcePolyfill)&&t.push("ce"),"content"in document.createElement("template")&&window.Promise&&Array.from&&document.createDocumentFragment().cloneNode()instanceof DocumentFragment||(t=["lite"]),t.length){var n,o="webcomponents-"+t.join("-")+".js";if(window.WebComponents.root)n=window.WebComponents.root+o;else{var c=document.querySelector('script[src*="'+e+'"]');n=c.src.replace(e,o)}var r=document.createElement("script");r.src=n,"loading"===document.readyState?document.write(r.outerHTML):document.head.appendChild(r)}else{var d=function(){requestAnimationFrame(function(){window.WebComponents.ready=!0,document.dispatchEvent(new CustomEvent("WebComponentsReady",{bubbles:!0}))})};"loading"!==document.readyState?d():document.addEventListener("readystatechange",function a(){d(),document.removeEventListener("readystatechange",a)})}}();
	</script>

	<link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Roboto">
	<link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Roboto+Mono">
	<link rel="import" href="/src/drone/drone-app.html">

	<style>
		html, body {
			padding:0px;
			margin:0px;
		}
	</style>
</head>
<body>
	<drone-app></drone-app>
</body>
</html>
`

// files/login.html
var login = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8"/>
  <meta content="width=device-width, initial-scale=1" name="viewport"/>
  <meta content="ie=edge" http-equiv="x-ua-compatible"/>
  <link href="https://fonts.googleapis.com/css?family=Roboto" rel="stylesheet"/>
  <link href="https://fonts.googleapis.com/css?family=Roboto+Mono" rel="stylesheet"/>
  <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet"/>
  <link href="/static/favicon.ico" rel="icon" type="image/x-icon"/>
  <link rel="stylesheet" href="/static/app.css" />
  <title>login | drone</title>
</head>
<body>
  <div class="mdl-grid">
    <div class="mdl-layout-spacer"></div>
    <div class="mdl-card">
      <form action="/authorize" method="post">
        <div class="mdl-textfield mdl-js-textfield">
          <input class="mdl-textfield__input" type="text" id="username" name="username" />
          <label class="mdl-textfield__label" for="username">Username</label>
        </div>
        <div class="mdl-textfield mdl-js-textfield">
          <input class="mdl-textfield__input" type="password" id="userpass" name="password" />
          <label class="mdl-textfield__label" for="userpass">Password</label>
        </div>
        <div class="mdl-dialog__actions">
          <input type="submit" class="mdl-button mdl-button--colored mdl-js-button" value="Login" />
        </div>
      </form>
    </div>
    <div class="mdl-layout-spacer"></div>
  </div>
  <script src="https://code.getmdl.io/1.1.3/material.min.js"></script>
</body>
</html>
`

// files/logout.html
var logout = `LOGOUT
`

// files/script.html
var script = `
		<script>
			window.ENV = {};
			window.ENV.server = window.location.protocol+"//"+window.location.host;
			{{ if .csrf }}window.ENV.csrf = "{{ .csrf }}"{{ end }}
			{{ if .user }}
			window.USER = {{ json .user }};
			{{ end }}
		</script>
`
