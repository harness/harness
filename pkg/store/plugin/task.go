package plugin

import (
	common "github.com/drone/drone/pkg/types"
)

type GetTaskReq struct {
	Repo  string
	Build int
	Task  int
}

type GetTaskResp struct {
	Task *common.Task
}

func (c *Client) GetTask(repo string, build int, task int) (*common.Task, error) {
	req := &GetTaskReq{repo, build, task}
	res := &GetTaskResp{}
	err := c.Call("Datastore.GetTask", req, res)
	return res.Task, err
}

type GetTaskLogsReq struct {
	Repo  string
	Build int
	Task  int
}

type GetTaskLogsResp struct {
	Logs []byte
}

func (c *Client) GetTaskLogs(repo string, build int, task int) ([]byte, error) {
	req := &GetTaskLogsReq{repo, build, task}
	res := &GetTaskLogsResp{}
	err := c.Call("Datastore.GetTaskLogs", req, res)
	return res.Logs, err
}

type GetTaskListReq struct {
	Repo  string
	Build int
}

type GetTaskListResp struct {
	Tasks []*common.Task
}

func (c *Client) GetTaskList(repo string, build int) ([]*common.Task, error) {
	req := &GetTaskListReq{repo, build}
	res := &GetTaskListResp{}
	err := c.Call("Datastore.GetTaskList", req, res)
	return res.Tasks, err
}

type UpsertTaskReq struct {
	Repo  string
	Build int
	Task  *common.Task
}

func (c *Client) UpsertTask(repo string, build int, task *common.Task) error {
	req := &UpsertTaskReq{repo, build, task}
	return c.Call("Datastore.UpsertTask", req, nil)
}

type UpsertTaskLogsReq struct {
	Repo  string
	Build int
	Task  int
	Logs  []byte
}

func (c *Client) UpsertTaskLogs(repo string, build int, task int, log []byte) error {
	req := &UpsertTaskLogsReq{repo, build, task, log}
	return c.Call("Datastore.UpsertTaskLogs", req, nil)
}
