package utils

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-logs/pkg/logs"
)

const (
	LocalCertDir = "/usr/local/share/ca-certificates/"
	// ClientSecret = "da9bbc28-01d8-43af-8c8a-fb0654937231"

	pathenvfile = "/usr/include/serial-dev"
)

var (
	RemoteBrokerURL string
	hostname        string
)

//SetHostname test name
func SetHostname(h string) {
	hostname = h
}

//Hostname get hostname
func Hostname() string {
	if len(hostname) > 0 {
		return hostname
	}
	envdev := make(map[string]string)
	if fileenv, err := os.Open(pathenvfile); err != nil {
		logs.LogWarn.Printf("error: reading file env, %s", err)
	} else {
		scanner := bufio.NewScanner(fileenv)
		for scanner.Scan() {
			line := scanner.Text()
			// log.Println(line)
			split := strings.Split(line, "=")
			if len(split) > 1 {
				envdev[split[0]] = split[1]
			}
		}
	}
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		logs.LogError.Fatalf("Error: there is not hostname! %s", err)
	}
	if v, ok := envdev["sn-dev"]; ok {
		reg, err := regexp.Compile("[^a-zA-Z0-9\\-_\\.]+")
		if err != nil {
			log.Println(err)
		}
		processdString := reg.ReplaceAllString(v, "")
		// log.Println(processdString)
		if len(processdString) > 0 {
			hostname = processdString
		}
	}
	return hostname

}

func LoadLocalCert(localCertDir string) *tls.Config {

	// Get the SystemCertPool, continue with an empty pool on error
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	// Read in the cert file
	certs, err := ioutil.ReadDir(localCertDir)
	if err != nil {
		log.Printf("Failed to append %q to RootCAs: %v", localCertDir, err)
	} else {
		for _, cert := range certs {
			file, err := ioutil.ReadFile(localCertDir + cert.Name())
			if err != nil {
				log.Fatalf("Failed to append %q to RootCAs: %v", cert, err)
			}
			// Append our cert to the system pool
			if ok := rootCAs.AppendCertsFromPEM(file); !ok {
				log.Println("No certs appended, using system certs only")
			}
		}
	}

	// Trust the augmented cert pool in our client
	config := &tls.Config{
		//InsecureSkipVerify: *insecure,
		RootCAs: rootCAs,
	}
	// tr := &http.Transport{}
	tr := &http.Transport{
		TLSClientConfig: config,
		// Dial: (&net.Dialer{
		// 	Timeout: 30 * time.Second,
		// }).Dial,
		// TLSHandshakeTimeout: 10 * time.Second,
		// Dial: (&net.Dialer{
		// 	Timeout:   30 * time.Second,
		// 	KeepAlive: 60 * time.Second,
		// }).Dial,
		// TLSHandshakeTimeout:   10 * time.Second,
		// ResponseHeaderTimeout: 10 * time.Second,
		// ExpectContinueTimeout: 3 * time.Second,
	}
	tr.TLSClientConfig = config

	return config

	/**

	// Uses local self-signed cert
	req := http.NewRequest(http.MethodGet, "https://localhost/api/version", nil)
	resp, err := client.Do(req)
	// Handle resp and err

	// Still works with host-trusted CAs!
	req = http.NewRequest(http.MethodGet, "https://example.com/", nil)
	resp, err = client.Do(req)
	// Handle resp and err
	/**/
}

type MsgTick struct{}

func tick(ctx actor.Context, timeout time.Duration, quit <-chan int) {
	rootctx := ctx.ActorSystem().Root
	self := ctx.Self()
	t1 := time.NewTicker(timeout)
	defer t1.Stop()
	for {
		select {
		case <-t1.C:
			rootctx.Send(self, &MsgTick{})
		case <-quit:
			return
		}
	}
}
