package internal

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type Project struct {
	Owner     string `json:"owner_user_name"`
	Name      string `json:"name"`
	DepotPath string `json:"depot_path"`
	HttpsURL  string `json:"https_url"`
	IsPublic  bool   `json:"is_public"`
	Icon      string `json:"icon"`
	Role      string `json:"current_user_role"`
}

type Depot struct {
	DefaultBranch string `json:"default_branch"`
}

type ProjectListData struct {
	Page      int        `json:"page"`
	PageSize  int        `json:"pageSize"`
	TotalPage int        `json:"totalPage"`
	TotalRow  int        `json:"totalRow"`
	List      []*Project `json:"list"`
}

func (c *Client) GetProject(globalKey, projectName string) (*Project, error) {
	u := fmt.Sprintf("/user/%s/project/%s", globalKey, projectName)
	resp, err := c.Get(u, nil)
	if err != nil {
		return nil, err
	}

	project := &Project{}
	err = json.Unmarshal(resp, project)
	if err != nil {
		return nil, APIClientErr{"fail to parse project data", u, err}
	}
	return project, nil
}

func (c *Client) GetDepot(globalKey, projectName string) (*Depot, error) {
	u := fmt.Sprintf("/user/%s/project/%s/git", globalKey, projectName)
	resp, err := c.Get(u, nil)
	if err != nil {
		return nil, err
	}

	depot := &Depot{}
	err = json.Unmarshal(resp, depot)
	if err != nil {
		return nil, APIClientErr{"fail to parse depot data", u, err}
	}
	return depot, nil
}

func (c *Client) GetProjectList() ([]*Project, error) {
	u := "/user/projects"
	resp, err := c.Get(u, nil)
	if err != nil {
		return nil, err
	}
	data := &ProjectListData{}
	err = json.Unmarshal(resp, data)
	if err != nil {
		return nil, APIClientErr{"fail to parse project list data", u, err}
	}
	if data.TotalPage == 1 {
		return data.List, nil
	}

	projectList := make([]*Project, 0)
	projectList = append(projectList, data.List...)
	for i := 2; i <= data.TotalPage; i++ {
		params := url.Values{}
		params.Set("page", fmt.Sprintf("%d", i))
		resp, err := c.Get(u, params)
		if err != nil {
			return nil, err
		}
		data := &ProjectListData{}
		err = json.Unmarshal(resp, data)
		if err != nil {
			return nil, APIClientErr{"fail to parse project list data", u, err}
		}
		projectList = append(projectList, data.List...)
	}
	return projectList, nil
}
