// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package badge

var (
	badgeSuccess = `<svg xmlns="http://www.w3.org/2000/svg" width="91" height="20"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><rect rx="3" width="91" height="20" fill="#555"/><rect rx="3" x="37" width="54" height="20" fill="#4c1"/><path fill="#4c1" d="M37 0h4v20h-4z"/><rect rx="3" width="91" height="20" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="15" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="14">build</text><text x="63" y="15" fill="#010101" fill-opacity=".3">success</text><text x="63" y="14">success</text></g></svg>`
	badgeFailure = `<svg xmlns="http://www.w3.org/2000/svg" width="83" height="20"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><rect rx="3" width="83" height="20" fill="#555"/><rect rx="3" x="37" width="46" height="20" fill="#e05d44"/><path fill="#e05d44" d="M37 0h4v20h-4z"/><rect rx="3" width="83" height="20" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="15" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="14">build</text><text x="59" y="15" fill="#010101" fill-opacity=".3">failure</text><text x="59" y="14">failure</text></g></svg>`
	badgeStarted = `<svg xmlns="http://www.w3.org/2000/svg" width="87" height="20"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><rect rx="3" width="87" height="20" fill="#555"/><rect rx="3" x="37" width="50" height="20" fill="#dfb317"/><path fill="#dfb317" d="M37 0h4v20h-4z"/><rect rx="3" width="87" height="20" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="15" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="14">build</text><text x="61" y="15" fill="#010101" fill-opacity=".3">started</text><text x="61" y="14">started</text></g></svg>`
	badgeError   = `<svg xmlns="http://www.w3.org/2000/svg" width="76" height="20"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><rect rx="3" width="76" height="20" fill="#555"/><rect rx="3" x="37" width="39" height="20" fill="#9f9f9f"/><path fill="#9f9f9f" d="M37 0h4v20h-4z"/><rect rx="3" width="76" height="20" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="15" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="14">build</text><text x="55.5" y="15" fill="#010101" fill-opacity=".3">error</text><text x="55.5" y="14">error</text></g></svg>`
	badgeNone    = `<svg xmlns="http://www.w3.org/2000/svg" width="75" height="20"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><rect rx="3" width="75" height="20" fill="#555"/><rect rx="3" x="37" width="38" height="20" fill="#9f9f9f"/><path fill="#9f9f9f" d="M37 0h4v20h-4z"/><rect rx="3" width="75" height="20" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="15" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="14">build</text><text x="55" y="15" fill="#010101" fill-opacity=".3">none</text><text x="55" y="14">none</text></g></svg>`
)
