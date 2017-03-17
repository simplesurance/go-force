package force

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

const (
	grantType    = "password"
	loginURI     = "https://login.salesforce.com/services/oauth2/token"
	testLoginURI = "https://test.salesforce.com/services/oauth2/token"

	invalidSessionErrorCode = "INVALID_SESSION_ID"
)

type forceOauth struct {
	AccessToken string `json:"access_token"`
	InstanceURL string `json:"instance_url"`
	ID          string `json:"id"`
	IssuedAt    string `json:"issued_at"`
	Signature   string `json:"signature"`

	clientID      string
	clientSecret  string
	refreshToken  string
	userName      string
	password      string
	securityToken string
	environment   string
}

func (oauth *forceOauth) Validate() error {
	if oauth == nil || len(oauth.InstanceURL) == 0 || len(oauth.AccessToken) == 0 {
		return fmt.Errorf("Invalid Force Oauth Object: %#v", oauth)
	}

	return nil
}

func (oauth *forceOauth) Expired(apiErrors APIErrors) bool {
	for _, err := range apiErrors {
		if err.ErrorCode == invalidSessionErrorCode {
			return true
		}
	}

	return false
}

func (oauth *forceOauth) Authenticate() error {
	payload := url.Values{
		"grant_type":    {grantType},
		"client_id":     {oauth.clientID},
		"client_secret": {oauth.clientSecret},
		"username":      {oauth.userName},
		"password":      {fmt.Sprintf("%v%v", oauth.password, oauth.securityToken)},
	}

	// Build Uri
	uri := loginURI
	if oauth.environment == "sandbox" {
		uri = testLoginURI
	}

	// Build Body
	body := strings.NewReader(payload.Encode())

	// Build Request
	req, err := http.NewRequest("POST", uri, body)
	bb := make([]byte, body.Len())
	_, _ = body.Read(bb)
	_, _ = body.Seek(0, 0)
	fmt.Printf("BBBBBBBBBBBBBBBBBBBBB:::::::: %+v\n", string(bb))

	if err != nil {
		return fmt.Errorf("Error creating authenitcation request: %v", err)
	}

	// Add Headers
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", responseType)

	fmt.Printf("SSSSSSSSSSSSSSSSSSSSSSAAAAAAAAAAAAAAAAAA:::: %+v\n", req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Error sending authentication request: %v", err)
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Error reading authentication response bytes: %v", err)
	}

	fmt.Printf("RRRRRRRRRRRRRRRRRRRR:::::::::::::::; %+v\nAAAAAAAAAAAAAAAAAAAAAAAA:::::: %+v\n", resp, string(respBytes))
	err = resp.Body.Close()
	if err != nil {
		return fmt.Errorf("Cannot close response body: %v", err)
	}

	// Attempt to parse response as a force.com api error
	apiError := &APIError{}
	if err := json.Unmarshal(respBytes, apiError); err == nil {
		// Check if api error is valid
		if apiError.Validate() {
			return apiError
		}
	}

	if err := json.Unmarshal(respBytes, oauth); err != nil {
		return fmt.Errorf("Unable to unmarshal authentication response: %v", err)
	}

	return nil
}
