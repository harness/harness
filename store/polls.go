package store

import (
	"github.com/drone/drone/model"
	"golang.org/x/net/context"
)

//PollStore pollstore
type PollStore interface {
	// Get gets a poll by unique repository ID.
	Get(poll *model.Poll) (*model.Poll, error)

	// Create creates a new poll.
	Create(*model.Poll) error

	// Update updates a repo poll.
	Update(*model.Poll) error

	// Delete deletes a repo poll.
	Delete(*model.Poll) error

	//GetPollList list all polls
	List() ([]*model.Poll, error)
}

//GetPoll get a poll by repository ID.
func GetPoll(c context.Context, poll *model.Poll) (*model.Poll, error) {
	return FromContext(c).Polls().Get(poll)
}

//CreatePoll creates a new repo poll.
func CreatePoll(c context.Context, poll *model.Poll) error {
	return FromContext(c).Polls().Create(poll)
}

//UpdatePoll updates a repo poll.
func UpdatePoll(c context.Context, poll *model.Poll) error {
	return FromContext(c).Polls().Update(poll)
}

//DeletePoll delete a repo poll.
func DeletePoll(c context.Context, poll *model.Poll) error {
	return FromContext(c).Polls().Delete(poll)
}

//GetPollList query all polls
func GetPollList(c context.Context) ([]*model.Poll, error) {
	return FromContext(c).Polls().List()
}
