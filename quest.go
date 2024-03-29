package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/3dw1nM0535/go-auth/handlers"
	"github.com/3dw1nM0535/go-auth/utils"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var clientid, clientsecret, host, port string
var conf *oauth2.Config
var state string

// User : retrieved and authenticated
type User struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Profile       string `json:"profile"`
	Picture       string `json:"picture"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Gender        string `json:"gender"`
}

// IndexHandler : handle index page
func IndexHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "index.tmpl", gin.H{})
}

// getLogin : get state from authentication url
func getLoginURL(state string) string {
	return conf.AuthCodeURL(state)
}

// LoginHandler : store token in session
func LoginHandler(c *gin.Context) {
	state := handlers.RandToken(32)
	session := sessions.Default(c)
	session.Set("state", state)
	log.Printf("Stored session: %v\n", state)
	session.Save()
	link := getLoginURL(state)
	c.HTML(http.StatusOK, "auth.tmpl", gin.H{"link": link})
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Panicf("Error loading ENV variables: " + err.Error())
	}
	clientid = utils.MustGet("ClientID")
	clientsecret = utils.MustGet("ClientSecret")
	host = utils.MustGet("SERVER_HOST")
	port = utils.MustGet("SERVER_PORT")

	conf = &oauth2.Config{
		ClientID:     clientid,
		ClientSecret: clientsecret,
		RedirectURL:  "http://localhost:9090/auth",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.email",
			"openid",
		},
		Endpoint: google.Endpoint,
	}
}

func authHandler(c *gin.Context) {
	// Check state validity
	session := sessions.Default(c)
	retrievedState := session.Get("state")
	if retrievedState != c.Request.URL.Query().Get("state") {
		c.HTML(http.StatusUnauthorized, "error.tmpl", gin.H{"message": "Invalid session state."})
		return
	}

	// Handle the exchange code to initiate transport
	code := c.Request.URL.Query().Get("code")
	token, err := conf.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Println(err)
		c.HTML(http.StatusBadRequest, "error.tmpl", code)
		return
	}
	// Construct the client
	client := conf.Client(oauth2.NoContext, token)
	userInfo, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		log.Println(err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	defer userInfo.Body.Close()
	data, _ := ioutil.ReadAll(userInfo.Body)
	log.Println(string(data))
	u := User{}
	if err = json.Unmarshal(data, &u); err != nil {
		log.Println(err)
		c.HTML(http.StatusBadRequest, "error.tmpl", gin.H{"message": err})
		return
	}

	session.Set("user-id", u.Email)
	if err = session.Save(); err != nil {
		log.Println(err)
		c.HTML(http.StatusBadRequest, "error.tmpl", gin.H{"message": err})
		return
	}

	c.HTML(http.StatusOK, "battle.tmpl", gin.H{"email": u.Email})

}

func main() {
	app := gin.New()
	store := sessions.NewCookieStore([]byte(handlers.RandToken(64)))
	store.Options(sessions.Options{
		Path:   "/",
		MaxAge: 86400 * 7,
	})
	app.Use(sessions.Sessions("goquestsession", store))
	app.Static("/css", "./static/css")
	app.Static("/img", "./static/img")
	app.LoadHTMLGlob("templates/*")

	app.GET("/", IndexHandler)
	app.GET("/login", LoginHandler)
	app.GET("/auth", authHandler)
	app.Run(host + ":" + port)
}
