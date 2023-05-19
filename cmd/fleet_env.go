//go:build !sibus
// +build !sibus

package main

const (
	keycloakurl_  = "https://fleet.nebulae.com.co/auth"
	clientid_     = "devices-nats"
	clientSecret_ = "f207031c-8384-432a-9ea5-ce9be8121712"
	redirecturl_  = "https://fleet-nats.nebulae.com.co"
	realm_        = "DEVICES"
	urlin_        = "https://fleet-nats.nebulae.com.co"
	remoteBroker_ = "wss://fleet-nats.nebulae.com.co:443"
)
