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

package fixtures

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler returns an http.Handler that is capable of handing a variety of mock
// Coding requests and returns mock responses.
func Handler() http.Handler {
	gin.SetMode(gin.TestMode)

	e := gin.New()
	e.POST("/api/oauth/access_token_v2", getToken)
	e.GET("/api/account/current_user", getUser)
	e.GET("/api/user/:gk/project/:prj", getProject)
	e.GET("/api/user/:gk/project/:prj/git", getDepot)
	e.GET("/api/user/:gk/project/:prj/git/blob/:ref/:path", getFile)
	e.GET("/api/user/:gk/project/:prj/git/hooks", getHooks)
	e.POST("/api/user/:gk/project/:prj/git/hook", postHook)
	e.PUT("/api/user/:gk/project/:prj/git/hook/:id", putHook)
	e.DELETE("/api/user/:gk/project/:prj/git/hook/:id", deleteHook)

	return e
}

func getToken(c *gin.Context) {
	c.Header("Content-Type", "application/json;charset=UTF-8")
	switch c.PostForm("grant_type") {
	case "refresh_token":
		switch c.PostForm("refresh_token") {
		case "i9i0HQqNR8bTY4rALYEF2itayFJNbnzC1eMFppwT":
			c.String(200, refreshedTokenPayload)
		default:
			c.String(200, invalidRefreshTokenPayload)
		}
	case "authorization_code":
		fallthrough
	default:
		switch c.PostForm("code") {
		case "code":
			c.String(200, tokenPayload)
		default:
			c.String(200, invalidCodePayload)
		}
	}
}

func getUser(c *gin.Context) {
	c.Header("Content-Type", "application/json;charset=UTF-8")
	switch c.Query("access_token") {
	case "KTNF2ALdm3ofbtxLh6IbV95Ro5AKWJUP":
		c.String(200, userPayload)
	default:
		c.String(200, userNotFoundPayload)
	}
}

func getProject(c *gin.Context) {
	c.Header("Content-Type", "application/json;charset=UTF-8")
	switch fmt.Sprintf("%s/%s", c.Param("gk"), c.Param("prj")) {
	case "demo1/test1":
		c.String(200, fakeProjectPayload)
	case "demo1/perm_owner":
		c.String(200, fakePermOwnerPayload)
	case "demo1/perm_admin":
		c.String(200, fakePermAdminPayload)
	case "demo1/perm_member":
		c.String(200, fakePermMemberPayload)
	case "demo1/perm_guest":
		c.String(200, fakePermGuestPayload)
	default:
		c.String(200, projectNotFoundPayload)
	}
}

func getDepot(c *gin.Context) {
	c.Header("Content-Type", "application/json;charset=UTF-8")
	switch fmt.Sprintf("%s/%s", c.Param("gk"), c.Param("prj")) {
	case "demo1/test1":
		c.String(200, fakeDepotPayload)
	default:
		c.String(200, projectNotFoundPayload)
	}
}

func getProjects(c *gin.Context) {
	c.Header("Content-Type", "application/json;charset=UTF-8")
	c.String(200, fakeProjectsPayload)
}

func getFile(c *gin.Context) {
	c.Header("Content-Type", "application/json;charset=UTF-8")
	switch fmt.Sprintf("%s/%s/%s/%s", c.Param("gk"), c.Param("prj"), c.Param("ref"), c.Param("path")) {
	case "demo1/test1/master/.drone.yml", "demo1/test1/4504a072cc/.drone.yml":
		c.String(200, fakeFilePayload)
	default:
		c.String(200, fileNotFoundPayload)
	}
}

func getHooks(c *gin.Context) {
	c.Header("Content-Type", "application/json;charset=UTF-8")
	c.String(200, fakeHooksPayload)
}

func postHook(c *gin.Context) {
	c.Header("Content-Type", "application/json;charset=UTF-8")
	switch c.PostForm("hook_url") {
	case "http://127.0.0.1":
		c.String(200, `{"code":0}`)
	default:
		c.String(200, `{"code":1}`)
	}
}

func putHook(c *gin.Context) {
	c.Header("Content-Type", "application/json;charset=UTF-8")
	switch c.Param("id") {
	case "2":
		c.String(200, `{"code":0}`)
	default:
		c.String(200, `{"code":1}`)
	}
}

func deleteHook(c *gin.Context) {
	c.Header("Content-Type", "application/json;charset=UTF-8")
	switch c.Param("id") {
	case "3":
		c.String(200, `{"code":0}`)
	default:
		c.String(200, `{"code":1}`)
	}
}

const tokenPayload = `
{
    "access_token":"KTNF2ALdm3ofbtxLh6IbV95Ro5AKWJUP",
    "refresh_token":"zVtxJrKhNhBcNyqCz1NggNAAmehAxnRO3Z0fXmCp",
    "expires_in":36000
}
`

const refreshedTokenPayload = `
{
    "access_token":"VDZupx0usVRV4oOd1FCu4xUxgk8SY0TK",
    "refresh_token":"BenBQq7TWZ7Cp0aUM47nQjTz2QHNmTWcPctB609n",
    "expires_in":36000
}
`

const invalidRefreshTokenPayload = `
{
    "code":3006,
    "msg":{
        "oauth_refresh_token_error":"Token校验失败"
    }
}
`

const invalidCodePayload = `
{
    "code":3003,
    "msg":{
        "oauth_validate_code_error":"code校验失败"
    }
}
`

const userPayload = `
{
    "code":0,
    "data":{
        "global_key":"demo1",
        "email":"demo1@gmail.com",
        "avatar":"/static/fruit_avatar/Fruit-20.png"
    }
}
`

const userNotFoundPayload = `
{
    "code":1,
    "msg":{
        "user_not_login":"用户未登录"
    }
}
`

const fakeProjectPayload = `
{
    "code":0,
    "data":{
        "owner_user_name":"demo1",
        "name":"test1",
        "depot_path":"/u/gilala/p/abp/git",
        "https_url":"https://git.coding.net/demo1/test1.git",
        "is_public": false,
        "icon":"/static/project_icon/scenery-5.png",
        "current_user_role":"owner"
    }
}
`

const fakePermOwnerPayload = `
{
    "code":0,
    "data":{
        "current_user_role":"owner"
    }
}
`

const fakePermAdminPayload = `
{
    "code":0,
    "data":{
        "current_user_role":"admin"
    }
}
`

const fakePermMemberPayload = `
{
    "code":0,
    "data":{
        "current_user_role":"member"
    }
}
`

const fakePermGuestPayload = `
{
    "code":0,
    "data":{
        "current_user_role":"guest"
    }
}
`

const fakeDepotPayload = `
{
    "code":0,
    "data":{
        "default_branch":"master"
    }
}
`

const projectNotFoundPayload = `
{
    "code":1100,
    "msg":{
        "project_not_exists":"项目不存在"
		}
}
`

const fakeProjectsPayload = `
{
    "code":0,
    "data":{
        "list":{
            "owner_user_name":"demo1",
            "name":"test1",
            "icon":"/static/project_icon/scenery-5.png",
        },
        "page":1,
        "pageSize":1,
        "totalPage":1,
        "totalRow":1
    }
}
`

const fakeFilePayload = `
{
    "code":0,
    "data":{
        "file":{
            "data":"pipeline:\n  test:\n    image: golang:1.6\n    commands:\n      - go test\n"
        }
    }
}
`

const fileNotFoundPayload = `
{
    "code":0,
    "data":{
        "ref":"master"
    }
}
`

const fakeHooksPayload = `
{
    "code":0,
    "data":[
        {
            "id":2,
            "hook_url":"http://127.0.0.2"
        },
        {
            "id":3,
            "hook_url":"http://127.0.0.3"
        }
    ]
}
`
