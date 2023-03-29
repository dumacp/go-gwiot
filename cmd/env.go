package main

import (
	"fmt"
	"os"
)

func getENV() {
	if len(keycloakurl) <= 0 {
		if len(os.Getenv("KEYCLOAK_URL_DEVICES")) > 0 {
			keycloakurl = os.Getenv("KEYCLOAK_URL_DEVICES")
		} else {
			keycloakurl = keycloakurl_
		}
	}
	if len(redirecturl) <= 0 {
		if len(os.Getenv("REDIRECT_URL_DEVICES")) > 0 {
			redirecturl = os.Getenv("REDIRECT_URL_DEVICES")
		} else {
			redirecturl = redirecturl_
		}
	}
	if len(remoteBroker) <= 0 {
		if len(os.Getenv("BROKER_URL_DEVICES")) > 0 {
			remoteBroker = os.Getenv("BROKER_URL_DEVICES")
		} else {
			remoteBroker = remoteBroker_
		}
	}
	if len(realm) <= 0 {
		if len(os.Getenv("REALM_DEVICES")) > 0 {
			realm = os.Getenv("REALM_DEVICES")
		} else {
			realm = realm_
		}
	}
	if len(clientSecret) <= 0 {
		if len(os.Getenv("CLIENTSECRET_DEVICES")) > 0 {
			clientSecret = os.Getenv("CLIENTSECRET_DEVICES")
		} else {
			clientSecret = clientSecret_
		}
	}
	if len(clientid) <= 0 {
		if len(os.Getenv("CLIENTID_DEVICES")) > 0 {
			clientid = os.Getenv("CLIENTID_DEVICES")
		} else {
			clientid = clientid_
		}
	}

	fmt.Printf("keycloakurl: %s\n", keycloakurl)
	fmt.Printf("redirecturl: %s\n", redirecturl)
	fmt.Printf("remoteMqttBrokerURL: %s\n", remoteBroker)
	fmt.Printf("realm: %s\n", realm)
	fmt.Printf("clientSecret: %s\n", clientSecret)
	fmt.Printf("clientid: %s\n", clientid)
}
