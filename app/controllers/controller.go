package controllers

import (
	"bytes"
	"html/template"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/kevingil/blog/app/models"
	"github.com/kevingil/blog/app/utils"
	"github.com/kevingil/blog/app/views"
	"golang.org/x/crypto/bcrypt"
)

type Context struct {
	User         *models.User
	Article      *models.Article
	Articles     []*models.Article
	Project      *models.Project
	Projects     []*models.Project
	Skill        *models.Project
	Skills       []*models.Skill
	Tags         []*models.Tag
	About        string
	Contact      string
	ArticleCount int
	DraftCount   int
	View         template.HTML
}

var data Context

// Sessions is a user sessions.
var Sessions map[string]*models.User

func getCookie(r *http.Request) *http.Cookie {
	cookie := &http.Cookie{
		Name:  "session",
		Value: "",
	}

	for _, c := range r.Cookies() {
		if c.Name == "session" {
			cookie.Value = c.Value
			break
		}
	}

	return cookie
}

func permission(w http.ResponseWriter, r *http.Request) {
	path := strings.Split(r.URL.Path, "/")[1]
	cookie := getCookie(r)

	switch path {
	case "dashboard":
		if Sessions[cookie.Value] == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		}
	case "login":
	case "register":
		if Sessions[cookie.Value] != nil {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		}
	}
}

// RenderWithLayout is a utility function to render a child template with a specified layout.
func Hx(w http.ResponseWriter, r *http.Request, l string, t string, data Context) {
	var response bytes.Buffer
	var child bytes.Buffer

	if err := views.Tmpl.ExecuteTemplate(&child, t, data); err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	data.View = template.HTML(child.String())

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, child.String())

	} else {
		if err := views.Tmpl.ExecuteTemplate(&response, l, data); err != nil {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, response.String())

	}

}

func Index(w http.ResponseWriter, r *http.Request) {

	data.About = models.About()
	data.Skills = models.Skills_Test()
	//data.Projects = models.HomeProjects()
	data.Projects = models.GetProjects()

	// Render the template using the utility function
	Hx(w, r, "main_layout", "home", data)
}

func HomeFeed(w http.ResponseWriter, r *http.Request) {
	data.Articles = models.Articles()
	for _, post := range data.Articles {
		post.Tags = post.FindTags()
	}
	isHTMXRequest := r.Header.Get("HX-Request") == "true"
	var tmpl string

	if isHTMXRequest {
		tmpl = "home_feed"
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}

	var response bytes.Buffer

	if err := views.Tmpl.ExecuteTemplate(&response, tmpl, data); err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, response.String())
}

// Login is a controller for users to log in.
func Login(w http.ResponseWriter, r *http.Request) {
	permission(w, r)

	switch r.Method {
	case http.MethodGet:
		var response bytes.Buffer
		if err := views.Tmpl.ExecuteTemplate(&response, "login.gohtml", nil); err != nil {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		io.WriteString(w, response.String())
	case http.MethodPost:
		user := &models.User{
			Email: r.FormValue("email"),
		}
		user = user.Find()

		if user.ID == 0 {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		err := bcrypt.CompareHashAndPassword(user.Password, []byte(r.FormValue("password")))
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		sessionID := uuid.New().String()
		cookie := &http.Cookie{
			Name:  "session",
			Value: sessionID,
		}
		Sessions[sessionID] = user

		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}

// Logout is a controller for users to log out.
func Logout(w http.ResponseWriter, r *http.Request) {
	cookie := getCookie(r)
	if Sessions[cookie.Value] != nil {
		delete(Sessions, cookie.Value)
	}

	cookie = &http.Cookie{
		Name:  "session",
		Value: "",
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Register is a controller to register a user.
func Register(w http.ResponseWriter, r *http.Request) {
	permission(w, r)

	switch r.Method {
	case http.MethodGet:
		var response bytes.Buffer
		if err := views.Tmpl.ExecuteTemplate(&response, "register.gohtml", nil); err != nil {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		io.WriteString(w, response.String())
	case http.MethodPost:
		user := &models.User{
			Name:  r.FormValue("name"),
			Email: r.FormValue("email"),
		}
		password, err := bcrypt.GenerateFromPassword([]byte(r.FormValue("password")), bcrypt.MinCost)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if err := utils.ValidateEmail(user.Email); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		user.Password = password
		user = user.Find()

		if user.ID == 0 {
			user = user.Create()
		}

		sessionID := uuid.New().String()
		cookie := &http.Cookie{
			Name:  "session",
			Value: sessionID,
		}
		Sessions[sessionID] = user

		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}
