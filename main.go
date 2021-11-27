package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)


type checks struct {
	url   string
	param string
}

type headerFlags []string

var (
	httpClient *http.Client
	specialChars = []string{"\"", "'", "<", ">"}
	delimiter = "|xxdelixx|"
	workerCount *int
	rate *int
	headers headerFlags
)

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"


func (i *headerFlags) String() string {
	var builder string
	for _,v := range *i{
		builder += v + delimiter
	}
	return builder
}

func (i *headerFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	workerCount = flag.Int("worker", 1, "amount of worker as an int")
	proxyFlag := flag.String("proxy", "", "dsocks5://<ip>:<port>")
	flag.Var(&headers, "header", "custom header, can be used multiple times")
	rate = flag.Int("rate", 0, "requests per second")
	flag.Parse()

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: time.Second,
		}).DialContext,
		MaxConnsPerHost: *workerCount,
	}

	if *proxyFlag != "" {
		prox5,err := url.Parse(*proxyFlag)
		if err != nil {
			return
		}
		transport.Proxy = http.ProxyURL(prox5)
	}

	if *rate != 0 {
		*rate = 1000/(*rate)
	}

	cr := func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	httpClient = &http.Client{
		Transport:     transport,
		CheckRedirect: cr,
	}

	checkQueue := make(chan checks)

	go func() {
		ns := bufio.NewScanner(os.Stdin)
		for ns.Scan() {
			// Make sure that there are no duplicate parameter values
			checkQueue <- checks{url: createUniqueParams(ns.Text())}
		}
		close(checkQueue)
		return
	}()

	initChecks := makePool(checkQueue, func(c checks, out chan checks) {
		reflected, err := reflectionCheck(c.url)
		if err != nil {
			return
		}
		if len(reflected) == 0 {
			//fmt.Println("Nothing reflective on "+ c.url)
			return
		}
		// target url with each parameter that reflects
		// url1 + param1, url1 + param2
		for _, param := range reflected {
			//fmt.Println("Will test url "+c.url+" with param "+param)
			fmt.Printf("[reflected] [%s] %s\n", c.url, param)
			out <- checks{c.url, param}
		}
	})

	done := makePool(initChecks, func(c checks, out chan checks) {
		for _, char := range specialChars {
			wasReflected, err := charCheck(c.url, c.param, "prefiiix"+char+"suffiiix")
			if err != nil {
				fmt.Fprintf(os.Stderr, "error from charCheck for url %s with param %s with %s: %s", c.url, c.param, char, err)
				continue
			}
			if wasReflected {
				fmt.Printf("[reflected] [%s] %s with %s\n", c.url, c.param, char)
			}
		}
	})
	<-done
}

func createUniqueParams(uri string) string {
	var present []string
	u, err := url.Parse(uri)
	if err != nil {
		log.Fatal(err)
	}
	q := u.Query()
	for key, values := range u.Query(){
		for _, val := range values {
			if !contains(present,val){
				present = append(present, val)
			}else {
				q.Set(key,createUniqueString(6))
			}
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func createUniqueString(length int) string{
	var seededRand *rand.Rand = rand.New(
		rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func reflectionCheck(target string) ([]string, error){
	result := make([]string, 0)

	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		return result, err
	}

	if headers.String() != ""{
		for _,item := range strings.Split(headers.String(),delimiter){
			if strings.Contains(item,":"){
				req.Header.Add(strings.TrimSpace(strings.Split(item,":")[0]),strings.TrimSpace(strings.Split(item,":")[1]))
			}
		}
	}
	// throttle by rate
	time.Sleep(time.Duration(*rate) * time.Millisecond)
	resp, err := httpClient.Do(req)
	if err != nil {
		return result, err
	}
	if resp.Body == nil {
		return result, err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	if strings.HasPrefix(resp.Status, "3") {
		return result, nil
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "" && !strings.Contains(ct, "html") {
		return result, nil
	}

	body := string(b)

	u, err := url.Parse(target)
	if err != nil {
		return result, err
	}

	// Actual check
	for key, values := range u.Query() {
		for _, val := range values {
			if !strings.Contains(body, val) {
				continue
			}
			result = append(result, key)
		}
	}
	return result, nil
}

func charCheck(target string, param string, suffix string) (bool,error){
	u, err := url.Parse(target)
	if err != nil {
		return false, err
	}

	qs := u.Query()
	val := qs.Get(param)

	qs.Set(param, val+suffix)
	u.RawQuery = qs.Encode()

	reflected, err := reflectionCheck(u.String())
	if err != nil {
		return false, err
	}

	for _, r := range reflected {
		if r == param {
			return true, nil
		}
	}

	return false, nil
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

type workerFunc func(checks, chan checks)

func makePool(input chan checks, fn workerFunc) chan checks{
	var wg sync.WaitGroup

	output := make(chan checks)
	for i := 0; i < *workerCount; i++ {
		wg.Add(1)
		go func() {
			for c := range input {
				fn(c, output)
			}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(output)
	}()

	return output
}