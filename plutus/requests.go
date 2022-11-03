package plutus

// this file will define various methods for making requests
import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sparrc/go-ping"
	"golang.org/x/net/proxy"
)

var (
	httpTransport    = &http.Transport{}
	torHTTPClient    = &http.Client{Transport: httpTransport, Timeout: 10 * time.Second}
	directHTTPClient = &http.Client{Timeout: 10 * time.Second}
	proxyAddr        = "127.0.0.1:9150"
)

func init() {
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err != nil {
		panic(err)
	}

	httpTransport.Dial = dialer.Dial
}

// TorGET makes a safe GET request over Tor
func TorGET(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Err: %s\n", err.Error())
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; rv:68.0) Gecko/20100101 Firefox/68.0")

	resp, err := torHTTPClient.Do(req)
	if err != nil {
		log.Printf("Err: %s\n", err.Error())
		return nil, errors.New("Could not request remote resource")
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Err: %s\n", err.Error())
		return nil, err
	}

	return body, nil
}

// Ping - This function will ping one of our services directly to ensure that it's alive
func Ping(url string) error {
	pinger, err := ping.NewPinger(url)
	if err != nil {
		return err
	}

	pinger.Count = 6
	pinger.Run()

	stats := pinger.Statistics()
	if stats.PacketsRecv != stats.PacketsSent {
		return fmt.Errorf("Could not reliably reach service at url: %s - sent %d packets, received %d in response", url, stats.PacketsSent, stats.PacketsRecv)
	}

	return nil
}

// TorJSONPOST posts json to an endpoint over Tor
func TorJSONPOST(url string, json []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(json))
	if err != nil {
		log.Printf("Err: %s\n", err.Error())
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; rv:68.0) Gecko/20100101 Firefox/68.0")
	req.Header.Add("Content-Type", "application/json")

	resp, err := torHTTPClient.Do(req)
	if err != nil {
		log.Printf("Err: %s\n", err.Error())
		return nil, errors.New("Could not request remote resource")
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Err: %s\n", err.Error())
		return nil, err
	}

	return body, nil
}

// DirectJSONPOST is used to hit RPC endpoints (they will be on localhost, hence the need for a direct connection. We can't (and it also doesn't make sense to) proxy through localhost(tor) just to come back around and hit localhost again)
func DirectJSONPOST(url string, json []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(json))
	if err != nil {
		log.Printf("Err: %s\n", err.Error())
		return nil, err
	}

	switch strings.Contains(url, "json_rpc") {
	case true:
		req.Header.Add("Content-Type", "application/json")
	case false:
		req.Header.Add("Content-Type", "text/plain")
	}

	resp, err := directHTTPClient.Do(req)
	if err != nil {
		log.Printf("Err: %s\n", err.Error())
		return nil, errors.New("Could not request remote resource")
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Err: %s\n", err.Error())
		return nil, err
	}

	return body, nil
}
