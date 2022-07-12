package tokenutil

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/pkg/errors"
)

const (
	// environment variable to override the default msi endpoint
	envMsiEndpoint = "MSI_ENDPOINT"
)

// RegistryRefreshToken is the response body from ACR exchange API
type RegistryRefreshToken struct {
	RefreshToken string `json:"refresh_token"`
}

// GetRegistryRefreshToken return a Registry token.
// It first gets an ARM token, and makes a POST request to registry/v2 endpoint to
// exchange it for a Registry token. Steps mentioned in detail below
// Authentication: https://github.com/Azure/acr/blob/master/docs/AAD-OAuth.md#authenticating-to-a-registry-with-azure-cli
// Exchange: https://github.com/Azure/acr/blob/master/docs/AAD-OAuth.md#calling-post-oauth2exchange-to-get-an-acr-refresh-token
// Note, we don't need to do token challenge part.
func GetRegistryRefreshToken(registry, resourceID, clientID string) (string, error) {
	armToken, err := GetRefreshAuthToken(resourceID, clientID)
	if err != nil {
		return "", errors.Wrap(err, "unable to get ARM token")
	}

	client := autorest.NewClientWithUserAgent("azure/acr/tasks")
	exchangeURL := fmt.Sprintf("https://%s/oauth2/exchange", registry)

	v := url.Values{}
	v.Set("grant_type", "access_token")
	v.Set("service", registry)
	v.Set("access_token", armToken.AccessToken)

	req, err := http.NewRequest("POST", exchangeURL, strings.NewReader(v.Encode()))
	if err != nil {
		return "", errors.Wrap(err, "unable to create the request to get ACR refresh token")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "unable to send the request to get ACR refresh token")
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get ACR refresh token, exchange API response code: %s", response.Status)
	}

	var token RegistryRefreshToken
	jsonResponse, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", errors.Wrap(err, "unable to read the response from ACR exchange API")
	}
	err = json.Unmarshal(jsonResponse, &token)
	if err != nil {
		return "", errors.Wrapf(err, "unable to parse the response from ACR exchange API: %s", jsonResponse)
	}
	return token.RefreshToken, nil
}

// GetRefreshAuthToken gets and refreshes an Auth token for the resourceID
func GetRefreshAuthToken(resourceID, clientID string) (*adal.Token, error) {
	spToken, err := GetServicePrincipalToken(resourceID, clientID)
	if err != nil {
		return nil, err
	}
	// try refresh
	if err := spToken.EnsureFresh(); err != nil {
		return nil, err
	}
	token := spToken.Token()
	return &token, nil
}

// GetServicePrincipalToken gets ServicePrincipal token
// it is based on github.com/Azure/acr-builder/vendor/github.com/Azure/go-autorest/autorest/azure/auth/auth.go and allows overriding the msi endpont using environment variable
func GetServicePrincipalToken(resourceID, clientID string) (*adal.ServicePrincipalToken, error) {
	mc := GetMSIConfig(resourceID, clientID)
	// default to the well known endpoint for getting MSI authentications tokens
	msiEndpoint := "http://169.254.169.254/metadata/identity/oauth2/token"

	// override the default from environment variable
	if endpoint := os.Getenv(envMsiEndpoint); endpoint != "" {
		msiEndpoint = endpoint
	}

	var spToken *adal.ServicePrincipalToken
	var err error
	if mc.ClientID == "" {
		spToken, err = adal.NewServicePrincipalTokenFromMSI(msiEndpoint, mc.Resource)
		if err != nil {
			return nil, fmt.Errorf("failed to get oauth token from MSI: %v", err)
		}
	} else {
		spToken, err = adal.NewServicePrincipalTokenFromMSIWithUserAssignedID(msiEndpoint, mc.Resource, mc.ClientID)
		if err != nil {
			return nil, fmt.Errorf("failed to get oauth token from MSI for user assigned identity: %v", err)
		}
	}

	return spToken, nil
}

// GetMSIConfig gets the MSI Config given resourceID and MSI clientID
func GetMSIConfig(resourceID, clientID string) *auth.MSIConfig {
	return &auth.MSIConfig{
		Resource: resourceID,
		ClientID: clientID,
	}
}
