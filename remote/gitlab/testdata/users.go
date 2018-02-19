// Copyright 2018 Drone.IO Inc.
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

package testdata

var currentUserPayload = []byte(`
{
  "id": 1,
  "username": "john_smith",
  "email": "john@example.com",
  "name": "John Smith",
  "private_token": "dd34asd13as",
  "state": "active",
  "created_at": "2012-05-23T08:00:58Z",
  "bio": null,
  "skype": "",
  "linkedin": "",
  "twitter": "",
  "website_url": "",
  "theme_id": 1,
  "color_scheme_id": 2,
  "is_admin": false,
  "can_create_group": true,
  "can_create_project": true,
  "projects_limit": 100
}
`)
