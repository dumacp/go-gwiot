package utils

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/dumacp/go-logs/pkg/logs"
)

func Get(client *http.Client,
	url, username, password string, jsonStr []byte) ([]byte, error) {
	body, _, err := GetWithCode(client, url, username, password, jsonStr, 3*time.Second)
	return body, err
}

func GetWithRetrytimeout(client *http.Client,
	url, username, password string, jsonStr []byte, timeout time.Duration) ([]byte, error) {
	body, _, err := GetWithCode(client, url, username, password, jsonStr, timeout)
	return body, err
}

func GetWithCode(client *http.Client,
	url, username, password string, jsonStr []byte, retrytimeout time.Duration) ([]byte, int, error) {
	req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	// fmt.Println(username, password)
	if len(username) > 0 || len(password) > 0 {
		req.SetBasicAuth(username, password)
	}

	if client == nil {
		tlsConfig := LoadLocalCert(LocalCertDir)
		tr := &http.Transport{
			TLSClientConfig: tlsConfig,
			Dial: (&net.Dialer{
				Timeout: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   20 * time.Second,
			ExpectContinueTimeout: 3 * time.Second,
		}
		client = &http.Client{Transport: tr}
		client.Timeout = 30 * time.Second
	}

	logs.LogBuild.Printf("Get request: %+v", req)
	var resp *http.Response
	rangex := make([]int, 3)
	for range rangex {
		resp, err = client.Do(req)
		if err != nil {
			time.Sleep(retrytimeout)
			continue
		}
		break
	}
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, resp.StatusCode, err
		}
		return nil, resp.StatusCode, fmt.Errorf("StatusCode: %d, resp: %s, req: %s", resp.StatusCode, body, req.URL)
	}
	if body, err := io.ReadAll(resp.Body); err != nil {
		return nil, resp.StatusCode, err
	} else {
		return body, resp.StatusCode, nil
	}
}
