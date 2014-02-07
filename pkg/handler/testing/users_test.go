package testing

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/drone/drone/database"
	. "github.com/drone/drone/database/testing"
	"github.com/drone/drone/handler"
	. "github.com/drone/drone/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestUserProfilePage(t *testing.T) {
	// seed the database with values
	Setup()
	defer Teardown()

	// dummy request
	req := http.Request{}
	req.Form = url.Values{}

	Convey("User Profile", t, func() {
		SkipConvey("View Profile Information", func() {
			user, _ := database.GetUser(1)
			res := httptest.NewRecorder()
			handler.UserUpdate(res, &req, user)

			Convey("Email Address is correct", func() {

			})
			Convey("User Name is correct", func() {

			})
		})
		Convey("Update Email Address", func() {
			Convey("With a Valid Email Address", func() {
				user, _ := database.GetUser(1)
				req.Form.Set("name", "John Smith")
				req.Form.Set("email", "John.Smith@gmail.com")
				res := httptest.NewRecorder()
				handler.UserUpdate(res, &req, user)

				So(res.Code, ShouldEqual, http.StatusOK)
			})
			Convey("With an Invalid Email Address", func() {
				user, _ := database.GetUser(1)
				req.Form.Set("name", "John Smith")
				req.Form.Set("email", "John.Smith")
				res := httptest.NewRecorder()
				handler.UserUpdate(res, &req, user)

				So(res.Code, ShouldEqual, http.StatusBadRequest)
				So(res.Body.String(), ShouldContainSubstring, ErrInvalidEmail.Error())
			})
			Convey("With an Empty Email Address", func() {
				user, _ := database.GetUser(1)
				req.Form.Set("name", "John Smith")
				req.Form.Set("email", "")
				res := httptest.NewRecorder()
				handler.UserUpdate(res, &req, user)

				So(res.Code, ShouldEqual, http.StatusBadRequest)
				So(res.Body.String(), ShouldContainSubstring, ErrInvalidEmail.Error())
			})
			Convey("With a Duplicate Email Address", func() {
				user, _ := database.GetUser(1)
				req.Form.Set("name", "John Smith")
				req.Form.Set("email", "cavepig@gmail.com")
				res := httptest.NewRecorder()
				handler.UserUpdate(res, &req, user)

				So(res.Code, ShouldEqual, http.StatusBadRequest)
			})
		})

		Convey("Update User Name", func() {
			Convey("With a Valid Name", func() {
				user, _ := database.GetUser(1)
				req.Form.Set("name", "John Smith")
				req.Form.Set("email", "John.Smith@gmail.com")
				res := httptest.NewRecorder()
				handler.UserUpdate(res, &req, user)

				So(res.Code, ShouldEqual, http.StatusOK)
			})
			Convey("With an Empty Name", func() {
				user, _ := database.GetUser(1)
				req.Form.Set("name", "")
				req.Form.Set("email", "John.Smith@gmail.com")
				res := httptest.NewRecorder()
				handler.UserUpdate(res, &req, user)

				So(res.Code, ShouldEqual, http.StatusBadRequest)
				So(res.Body.String(), ShouldContainSubstring, ErrInvalidUserName.Error())
			})
		})

		Convey("Change Password", func() {
			Convey("To a Valid Password", func() {
				user, _ := database.GetUser(1)
				req.Form.Set("password", "password123")
				res := httptest.NewRecorder()
				handler.UserPassUpdate(res, &req, user)

				So(res.Code, ShouldEqual, http.StatusOK)
				So(user.ComparePassword("password123"), ShouldBeNil)
			})
			Convey("To an Invalid Password, too short", func() {
				user, _ := database.GetUser(1)
				req.Form.Set("password", "123")
				res := httptest.NewRecorder()
				handler.UserPassUpdate(res, &req, user)

				So(res.Code, ShouldEqual, http.StatusBadRequest)
			})
		})
		Convey("Delete the Account", func() {
			Convey("Providing an Invalid Password", func() {
				user, _ := database.GetUser(1)
				req.Form.Set("password", "password111")
				res := httptest.NewRecorder()
				handler.UserDelete(res, &req, user)

				So(res.Code, ShouldEqual, http.StatusBadRequest)
			})
			SkipConvey("Providing a Valid Password", func() {
				// TODO Skipping because there are no teampltes
				// loaded which will cause a panic
				user, _ := database.GetUser(2)
				req.Form.Set("password", "password")
				res := httptest.NewRecorder()
				handler.UserDelete(res, &req, user)

				So(res.Code, ShouldEqual, http.StatusOK)
			})
		})
	})
}

func TestUserTeamPage(t *testing.T) {
	// seed the database with values
	Setup()
	defer Teardown()

	// dummy request
	//req := http.Request{}
	//req.Form = url.Values{}

	Convey("User Team Page", t, func() {
		SkipConvey("View List of Teams", func() {

		})
		SkipConvey("View Empty List of Teams", func() {

		})
	})

	Convey("Create a Team", t, func() {
		SkipConvey("With an Invalid Name", func() {

		})
		SkipConvey("With an Invalid Email", func() {

		})
		SkipConvey("With a Valid Name and Email", func() {

		})
	})
}
