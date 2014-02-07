package testing

import (
	//"net/http"
	//"net/http/httptest"
	//"net/url"
	"testing"

	//"github.com/drone/drone/database"
	. "github.com/drone/drone/database/testing"
	//"github.com/drone/drone/handler"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTeamProfilePage(t *testing.T) {
	// seed the database with values
	Setup()
	defer Teardown()

	// dummy request
	//req := http.Request{}
	//req.Form = url.Values{}

	Convey("Team Profile Page", t, func() {
		Convey("View Profile Information", func() {

			SkipConvey("Email Address is correct", func() {

			})
			SkipConvey("Team Name is correct", func() {

			})
			SkipConvey("GitHub Login is correct", func() {

			})
			SkipConvey("Bitbucket Login is correct", func() {

			})
		})
		Convey("Update Email Address", func() {
			SkipConvey("With a Valid Email Address", func() {

			})
			SkipConvey("With an Invalid Email Address", func() {

			})
			SkipConvey("With an Empty Email Address", func() {

			})
		})

		Convey("Update Team Name", func() {
			SkipConvey("With a Valid Name", func() {

			})
			SkipConvey("With an Invalid Name", func() {

			})
			SkipConvey("With an Empty Name", func() {

			})
		})

		Convey("Delete the Team", func() {
			SkipConvey("Providing an Invalid Password", func() {

			})
			SkipConvey("Providing a Valid Password", func() {

			})
		})
	})
}

func TestTeamMembersPage(t *testing.T) {
	// seed the database with values
	Setup()
	defer Teardown()

	// dummy request
	//req := http.Request{}
	//req.Form = url.Values{}

	Convey("Team Members Page", t, func() {
		SkipConvey("View List of Team Members", func() {

		})
		SkipConvey("Add a New Team Member", func() {

		})

		Convey("Edit a Team Member", func() {
			SkipConvey("Modify the Role", func() {

			})
			SkipConvey("Change to an Invalid Role", func() {

			})
			SkipConvey("Change from Owner to Read", func() {

			})
		})

		Convey("Delete a Team Member", func() {
			SkipConvey("Delete a Read-only Member", func() {

			})
			SkipConvey("Delete the Last Member", func() {

			})
			SkipConvey("Delete the Owner", func() {

			})
		})

		Convey("Accept Membership", func() {
			SkipConvey("Valid Invitation", func() {

			})
			SkipConvey("Expired Invitation", func() {

			})
			SkipConvey("Invalid or Forged Invitation", func() {

			})
		})
	})
}

func TestDashboardPage(t *testing.T) {
	// seed the database with values
	Setup()
	defer Teardown()

	// dummy request
	//req := http.Request{}
	//req.Form = url.Values{}

	SkipConvey("Team Dashboard", t, func() {

	})

	SkipConvey("User Dashboard", t, func() {

	})

	SkipConvey("Repo Dashboard", t, func() {

	})

	SkipConvey("Repo Settings", t, func() {

	})

	SkipConvey("Commit Dashboard", t, func() {

	})

	Convey("User Account", t, func() {
		SkipConvey("Login", func() {

		})
		SkipConvey("Logout", func() {

		})
		SkipConvey("Register", func() {

		})
		SkipConvey("Sign Up", func() {

		})
	})
}
