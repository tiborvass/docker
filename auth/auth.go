package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
)

// Where we store the config file
const CONFIGFILE = ".dockercfg"

// the registry server we want to login against
const INDEX_SERVER = "https://index.docker.io/v1"

type AuthConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	rootPath string
}

func NewAuthConfig(username, password, email, rootPath string) *AuthConfig {
	return &AuthConfig{
		Username: username,
		Password: password,
		Email:    email,
		rootPath: rootPath,
	}
}

func IndexServerAddress() string {
	if os.Getenv("DOCKER_INDEX_URL") != "" {
		return os.Getenv("DOCKER_INDEX_URL") + "/v1"
	}
	return INDEX_SERVER
}

// create a base64 encoded auth string to store in config
func EncodeAuth(authConfig *AuthConfig) string {
	authStr := authConfig.Username + ":" + authConfig.Password
	msg := []byte(authStr)
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(msg)))
	base64.StdEncoding.Encode(encoded, msg)
	return string(encoded)
}

// decode the auth string
func DecodeAuth(authStr string) (*AuthConfig, error) {
	decLen := base64.StdEncoding.DecodedLen(len(authStr))
	decoded := make([]byte, decLen)
	authByte := []byte(authStr)
	n, err := base64.StdEncoding.Decode(decoded, authByte)
	if err != nil {
		return nil, err
	}
	if n > decLen {
		return nil, fmt.Errorf("Something went wrong decoding auth config")
	}
	arr := strings.Split(string(decoded), ":")
	if len(arr) != 2 {
		return nil, fmt.Errorf("Invalid auth configuration file")
	}
	password := strings.Trim(arr[1], "\x00")
	return &AuthConfig{Username: arr[0], Password: password}, nil

}

// load up the auth config information and return values
// FIXME: use the internal golang config parser
func LoadConfig(rootPath string) (*AuthConfig, error) {
	confFile := path.Join(rootPath, CONFIGFILE)
	if _, err := os.Stat(confFile); err != nil {
		return &AuthConfig{}, fmt.Errorf("The Auth config file is missing")
	}
	b, err := ioutil.ReadFile(confFile)
	if err != nil {
		return nil, err
	}
	arr := strings.Split(string(b), "\n")
	if len(arr) < 2 {
		return nil, fmt.Errorf("The Auth config file is empty")
	}
	origAuth := strings.Split(arr[0], " = ")
	origEmail := strings.Split(arr[1], " = ")
	authConfig, err := DecodeAuth(origAuth[1])
	if err != nil {
		return nil, err
	}
	authConfig.Email = origEmail[1]
	authConfig.rootPath = rootPath
	return authConfig, nil
}

// save the auth config
func saveConfig(rootPath, authStr string, email string) error {
	confFile := path.Join(rootPath, CONFIGFILE)
	if len(email) == 0 {
		os.Remove(confFile)
		return nil
	}
	lines := "auth = " + authStr + "\n" + "email = " + email + "\n"
	b := []byte(lines)
	err := ioutil.WriteFile(confFile, b, 0600)
	if err != nil {
		return err
	}
	return nil
}

// try to register/login to the registry server
func Login(authConfig *AuthConfig) (string, error) {
	storeConfig := false
	client := &http.Client{}
	reqStatusCode := 0
	var status string
	var reqBody []byte
	jsonBody, err := json.Marshal(authConfig)
	if err != nil {
		return "", fmt.Errorf("Config Error: %s", err)
	}

	// using `bytes.NewReader(jsonBody)` here causes the server to respond with a 411 status.
	b := strings.NewReader(string(jsonBody))
	req1, err := http.Post(IndexServerAddress()+"/users/", "application/json; charset=utf-8", b)
	if err != nil {
		return "", fmt.Errorf("Server Error: %s", err)
	}
	reqStatusCode = req1.StatusCode
	defer req1.Body.Close()
	reqBody, err = ioutil.ReadAll(req1.Body)
	if err != nil {
		return "", fmt.Errorf("Server Error: [%#v] %s", reqStatusCode, err)
	}

	if reqStatusCode == 201 {
		status = "Account created. Please use the confirmation link we sent" +
			" to your e-mail to activate it.\n"
		storeConfig = true
	} else if reqStatusCode == 403 {
		return "", fmt.Errorf("Login: Your account hasn't been activated. " +
			"Please check your e-mail for a confirmation link.")
	} else if reqStatusCode == 400 {
		if string(reqBody) == "\"Username or email already exists\"" {
			req, err := http.NewRequest("GET", IndexServerAddress()+"/users/", nil)
			req.SetBasicAuth(authConfig.Username, authConfig.Password)
			resp, err := client.Do(req)
			if err != nil {
				return "", err
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return "", err
			}
			if resp.StatusCode == 200 {
				status = "Login Succeeded\n"
				storeConfig = true
			} else if resp.StatusCode == 401 {
				saveConfig(authConfig.rootPath, "", "")
				return "", fmt.Errorf("Wrong login/password, please try again")
			} else {
				return "", fmt.Errorf("Login: %s (Code: %d; Headers: %s)", body,
					resp.StatusCode, resp.Header)
			}
		} else {
			return "", fmt.Errorf("Registration: %s", reqBody)
		}
	} else {
		return "", fmt.Errorf("Unexpected status code [%d] : %s", reqStatusCode, reqBody)
	}
	if storeConfig {
		authStr := EncodeAuth(authConfig)
		saveConfig(authConfig.rootPath, authStr, authConfig.Email)
	}
	return status, nil
}
