package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dchest/authcookie"
	"github.com/drone/drone/pkg/database"
	"github.com/drone/drone/pkg/mail"
	. "github.com/drone/drone/pkg/model"
)

// Display a list of Team Members.
func TeamMembers(w http.ResponseWriter, r *http.Request, u *User) error {
	teamParam := r.FormValue(":team")
	team, err := database.GetTeamSlug(teamParam)
	if err != nil {
		return err
	}
	// user must be a team member admin
	if member, _ := database.IsMemberAdmin(u.ID, team.ID); !member {
		return fmt.Errorf("Forbidden")
	}
	members, err := database.ListMembers(team.ID)
	if err != nil {
		return err
	}
	data := struct {
		User    *User
		Team    *Team
		Members []*Member
	}{u, team, members}
	return RenderTemplate(w, "team_members.html", &data)
}

// Return an HTML form for creating a new Team Member.
func TeamMemberAdd(w http.ResponseWriter, r *http.Request, u *User) error {
	teamParam := r.FormValue(":team")
	team, err := database.GetTeamSlug(teamParam)
	if err != nil {
		return err
	}
	if member, _ := database.IsMemberAdmin(u.ID, team.ID); !member {
		return fmt.Errorf("Forbidden")
	}
	data := struct {
		User *User
		Team *Team
	}{u, team}
	return RenderTemplate(w, "members_add.html", &data)
}

// Return an HTML form for editing a Team Member.
func TeamMemberEdit(w http.ResponseWriter, r *http.Request, u *User) error {
	teamParam := r.FormValue(":team")
	team, err := database.GetTeamSlug(teamParam)
	if err != nil {
		return err
	}
	if member, _ := database.IsMemberAdmin(u.ID, team.ID); !member {
		return fmt.Errorf("Forbidden")
	}

	// get the ID from the URL parameter
	idstr := r.FormValue("id")
	id, err := strconv.Atoi(idstr)
	if err != nil {
		return err
	}

	user, err := database.GetUser(int64(id))
	if err != nil {
		return err
	}
	member, err := database.GetMember(user.ID, team.ID)
	if err != nil {
		return err
	}
	data := struct {
		User   *User
		Team   *Team
		Member *Member
	}{u, team, member}
	return RenderTemplate(w, "members_edit.html", &data)
}

// Update a specific Team Member.
func TeamMemberUpdate(w http.ResponseWriter, r *http.Request, u *User) error {
	roleParam := r.FormValue("Role")
	teamParam := r.FormValue(":team")

	// get the team from the database
	team, err := database.GetTeamSlug(teamParam)
	if err != nil {
		return RenderError(w, err, http.StatusNotFound)
	}
	// verify the user is a admin member of the team
	if member, _ := database.IsMemberAdmin(u.ID, team.ID); !member {
		return fmt.Errorf("Forbidden")
	}

	// get the ID from the URL parameter
	idstr := r.FormValue("id")
	id, err := strconv.Atoi(idstr)
	if err != nil {
		return err
	}

	// get the user from the database
	user, err := database.GetUser(int64(id))
	if err != nil {
		return RenderError(w, err, http.StatusNotFound)
	}

	// add the user to the team
	if err := database.SaveMember(user.ID, team.ID, roleParam); err != nil {
		return RenderError(w, err, http.StatusInternalServerError)
	}

	return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}

// Delete a specific Team Member.
func TeamMemberDelete(w http.ResponseWriter, r *http.Request, u *User) error {
	// get the team from the database
	teamParam := r.FormValue(":team")
	team, err := database.GetTeamSlug(teamParam)
	if err != nil {
		return RenderNotFound(w)
	}

	if member, _ := database.IsMemberAdmin(u.ID, team.ID); !member {
		return fmt.Errorf("Forbidden")
	}

	// get the ID from the URL parameter
	idstr := r.FormValue("id")
	id, err := strconv.Atoi(idstr)
	if err != nil {
		return err
	}

	// get the user from the database
	user, err := database.GetUser(int64(id))
	if err != nil {
		return RenderNotFound(w)
	}
	// must be at least 1 member
	members, err := database.ListMembers(team.ID)
	if err != nil {
		return err
	} else if len(members) == 1 {
		return fmt.Errorf("There must be at least 1 member per team")
	}
	// delete the member
	database.DeleteMember(user.ID, team.ID)
	http.Redirect(w, r, fmt.Sprintf("/account/team/%s/members", team.Name), http.StatusSeeOther)
	return nil
}

// Invite a new Team Member.
func TeamMemberInvite(w http.ResponseWriter, r *http.Request, u *User) error {
	teamParam := r.FormValue(":team")
	mailParam := r.FormValue("email")
	team, err := database.GetTeamSlug(teamParam)
	if err != nil {
		return RenderError(w, err, http.StatusNotFound)
	}
	if member, _ := database.IsMemberAdmin(u.ID, team.ID); !member {
		return fmt.Errorf("Forbidden")
	}

	// generate a token that is valid for 3 days to join the team
	token := authcookie.New(team.Name, time.Now().Add(72*time.Hour), secret)

	// hostname from settings
	hostname := database.SettingsMust().URL().String()
	emailEnabled := database.SettingsMust().SmtpServer != ""

	if !emailEnabled {
		// Email is not enabled, so must let the user know the signup link
		link := fmt.Sprintf("%v/accept?token=%v", hostname, token)
		return RenderText(w, link, http.StatusOK)
	}

	// send the invitation
	data := struct {
		User  *User
		Team  *Team
		Token string
		Host  string
	}{u, team, token, hostname}

	// send email async
	go mail.SendInvitation(team.Name, mailParam, &data)

	return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}

func TeamMemberAccept(w http.ResponseWriter, r *http.Request, u *User) error {
	// get the team name from the token
	token := r.FormValue("token")
	teamName := authcookie.Login(token, secret)
	if len(teamName) == 0 {
		return ErrInvalidTeamName
	}

	// get the team from the database
	// TODO it might make more sense to use the ID in case the Slug changes
	team, err := database.GetTeamSlug(teamName)
	if err != nil {
		return RenderError(w, err, http.StatusNotFound)
	}

	// add the user to the team.
	// by default the user has write access to the team, which means
	// they can add and manage new repositories.
	if err := database.SaveMember(u.ID, team.ID, RoleWrite); err != nil {
		return RenderError(w, err, http.StatusInternalServerError)
	}

	// send the user to the dashboard
	http.Redirect(w, r, "/dashboard/team/"+team.Name, http.StatusSeeOther)
	return nil
}
