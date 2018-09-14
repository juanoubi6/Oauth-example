package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"io/ioutil"
	"net/http"
	"strings"
	"errors"
)

var oauthStateString = "pseudo-random"
var projectId = "YOUR-GOOGLE-PROJECT-ID"
var adminRoleName = "YOUR-CUSTOM-ADMIN-NAME"

var googleOauthConfig = &oauth2.Config{
	RedirectURL:  "http://localhost:5000/oauth/callback",
	ClientSecret: "YOUR-CLIENT-SECRET",
	ClientID:     "YOUR-CLIENT-ID",
	Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
	Endpoint:     google.Endpoint,
}

func main() {
	http.HandleFunc("/", handleMain)
	http.HandleFunc("/login", handleGoogleLogin)
	http.HandleFunc("/oauth/callback", handleGoogleCallback)
	fmt.Println(http.ListenAndServe(":5000", nil))
}

func handleMain(w http.ResponseWriter, r *http.Request) {
	var htmlIndex = `<html><body><a href="/login">Google Log In</a></body></html>`
	fmt.Fprintf(w, htmlIndex)
}

func handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	url := googleOauthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

type GoogleUserData struct {
	Name     string `json:"family_name"`
	LastName string `json:"given_name"`
	Email    string `json:"email"`
	GoogleID string `json:"id"`
	Picture  string `json:"picture"`
}

type GoogleProjectData struct {
	Version  int    `json:"version"`
	Etag     string `json:"etag"`
	Bindings []struct {
		Role    string `json:"role"`
		Members []string`json:"members"`
	} `json:"bindings"`
}

type GoogleError struct{
	Error struct{
		Code int `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func handleGoogleCallback(w http.ResponseWriter, r *http.Request) {

	state := r.FormValue("state")
	code := r.FormValue("code")

	if state != oauthStateString {
		println("invalid oauth state")
		return
	}

	token, err := googleOauthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		println("code exchange failed: %s", err.Error())
	}

	//Map user information
	userInfo, err := getUserInfo(token)
	if err != nil {
		fmt.Println("error getting user info: " + err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	var userData GoogleUserData
	err = json.Unmarshal(userInfo, &userData)
	if err != nil {
		fmt.Println("error mapping user data: " + err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	//Map project information
	projectInfo, err := getProjectInfo()
	if err != nil {
		fmt.Println("error getting project info: " + err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	var projectData GoogleProjectData
	err = json.Unmarshal(projectInfo, &projectData)
	if err != nil {
		fmt.Println("error mapping project data: " + err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	isAdmin,err := checkIfUserIsAdmin(userData.Email,projectData)
	if err != nil {
		fmt.Println("error: " + err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	if isAdmin == true{
		fmt.Fprintf(w, "Content: %s\n", "I'm an admin")
	}else{
		fmt.Fprintf(w, "Content: %s\n", "I'm not an admin")
	}

}

func getUserInfo(token *oauth2.Token) ([]byte, error) {

	userInfoResponse, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed getting user info: %s", err.Error())
	}
	defer userInfoResponse.Body.Close()

	contents, err := ioutil.ReadAll(userInfoResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading user info response body: %s", err.Error())
	}

	return contents, nil
}

func getProjectInfo() ([]byte, error) {

	jsonFile,err := ioutil.ReadFile("serviceAccountJson.json")
	if err != nil {
		return nil, fmt.Errorf("failed reading account service json: %s", err.Error())
	}

	conf,err := google.JWTConfigFromJSON(jsonFile,"https://www.googleapis.com/auth/cloudplatformprojects")
	if err != nil {
		return nil, fmt.Errorf("failed creating account service config: %s", err.Error())
	}

	client := conf.Client(oauth2.NoContext)
	req, _ := http.NewRequest("POST", "https://cloudresourcemanager.googleapis.com/v1/projects/"+projectId+":getIamPolicy", nil)
	projectInfoResponse, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed getting project info: %s", err.Error())
	}
	defer projectInfoResponse.Body.Close()

	contents, err := ioutil.ReadAll(projectInfoResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body: %s", err.Error())
	}

	if projectInfoResponse.StatusCode >= 400{
		var googleError GoogleError
		err = json.Unmarshal(contents, &googleError)
		if err != nil {
			fmt.Println("error mapping google error: " + err.Error())
		}
		return nil, fmt.Errorf(googleError.Error.Message)
	}

	return contents, nil
}

//Checks if the user email is included in the email list of the admin role name we defined
func checkIfUserIsAdmin(userEmail string, projectData GoogleProjectData)(bool,error){

	var isAdmin bool = false

	if len(projectData.Bindings) == 0{
		return false,errors.New("No user data could be brought")
	}

	for _, binding := range projectData.Bindings{
		bindingData := strings.Split(binding.Role,"/")
		if bindingData[len(bindingData)-1] == adminRoleName{
			for _, member := range binding.Members{
				memberData := strings.Split(member,":")
				if memberData[0] == "user" && memberData[len(memberData)-1] == userEmail{
					isAdmin = true
					break
				}
			}
		}
	}

	return isAdmin,nil


}
