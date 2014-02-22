package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/drone/drone/pkg/database"
	. "github.com/drone/drone/pkg/model"
)

// Display a specific Team.
func TeamShow(w http.ResponseWriter, r *http.Request, u *User) error {
	teamParam := r.FormValue(":team")
	team, err := database.GetTeamSlug(teamParam)
	if err != nil {
		return err
	}
	if member, _ := database.IsMember(u.ID, team.ID); !member {
		return fmt.Errorf("Forbidden")
	}
	// list of repositories owned by Team
	repos, err := database.ListReposTeam(team.ID)
	if err != nil {
		return err
	}
	// list all user teams
	teams, err := database.ListTeams(u.ID)
	if err != nil {
		return err
	}
	// list of recent commits
	commits, err := database.ListCommitsTeam(team.ID)
	if err != nil {
		return err
	}
	data := struct {
		User    *User
		Team    *Team
		Teams   []*Team
		Repos   []*Repo
		Commits []*RepoCommit
	}{u, team, teams, repos, commits}
	return RenderTemplate(w, "team_dashboard.html", &data)
}

// Return an HTML form for editing a Team.
func TeamEdit(w http.ResponseWriter, r *http.Request, u *User) error {
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
	return RenderTemplate(w, "team_profile.html", &data)
}

// Return an HTML form for creating a Team.
func TeamAdd(w http.ResponseWriter, r *http.Request, u *User) error {
	return RenderTemplate(w, "user_teams_add.html", struct{ User *User }{u})
}

// Create a new Team.
func TeamCreate(w http.ResponseWriter, r *http.Request, u *User) error {
	// set the name and email from the form data
	team := Team{}
	team.SetName(r.FormValue("name"))
	team.SetEmail(r.FormValue("email"))

	if err := team.Validate(); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	}
	if err := database.SaveTeam(&team); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	}

	// add default member to the team (me)
	if err := database.SaveMember(u.ID, team.ID, RoleOwner); err != nil {
		return RenderError(w, err, http.StatusInternalServerError)
	}

	return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}

// Update a specific Team.
func TeamUpdate(w http.ResponseWriter, r *http.Request, u *User) error {
	// get team from the database
	teamName := r.FormValue(":team")
	team, err := database.GetTeamSlug(teamName)
	if err != nil {
		return fmt.Errorf("Forbidden")
	}
	if member, _ := database.IsMemberAdmin(u.ID, team.ID); !member {
		return fmt.Errorf("Forbidden")
	}

	team.Name = r.FormValue("name")
	team.SetEmail(r.FormValue("email"))

	if err := team.Validate(); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	}
	if err := database.SaveTeam(team); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	}

	return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}

// Delete Confirmation Page
func TeamDeleteConfirm(w http.ResponseWriter, r *http.Request, u *User) error {
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
	return RenderTemplate(w, "team_delete.html", &data)
}

// Delete a specific Team.
func TeamDelete(w http.ResponseWriter, r *http.Request, u *User) error {
	// get the team from the database
	teamParam := r.FormValue(":team")
	team, err := database.GetTeamSlug(teamParam)
	if err != nil {
		return RenderNotFound(w)
	}
	if member, _ := database.IsMemberAdmin(u.ID, team.ID); !member {
		return fmt.Errorf("Forbidden")
	}
	// the user must confirm their password before deleting
	password := r.FormValue("password")
	if err := u.ComparePassword(password); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	}

	database.DeleteTeam(team.ID)
	http.Redirect(w, r, "/account/user/teams", http.StatusSeeOther)
	return nil
}

// Wall display for the team
func TeamWall(w http.ResponseWriter, r *http.Request, u *User) error {
	teamParam := r.FormValue(":team")
	team, err := database.GetTeamSlug(teamParam)
	if err != nil {
		return err
	}

	if member, _ := database.IsMember(u.ID, team.ID); !member {
		return fmt.Errorf("Forbidden")
	}

	// list of recent commits
	commits, err := database.ListCommitsTeam(team.ID)
	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(commits)
}

// API endpoint for fetching the initial wall display data via AJAX
func TeamWallData(w http.ResponseWriter, r *http.Request, u *User) error {
	teamParam := r.FormValue(":team")
	team, err := database.GetTeamSlug(teamParam)
	if err != nil {
		return err
	}

	if member, _ := database.IsMember(u.ID, team.ID); !member {
		return fmt.Errorf("Forbidden")
	}

	// list of recent commits
	commits, err := database.ListCommitsTeam(team.ID)
	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(commits)
}
