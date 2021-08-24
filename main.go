package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

type Request struct {
	Index    int
	Response *http.Response
	Err      error
}

type Report struct {
	TotalRequests int
	TotalSuccess  int
	TotalFail     int
}

func exitGracefully(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

func check(err error) {
	if err != nil {
		exitGracefully(err)
	}
}

func doRequest(wg *sync.WaitGroup, resultC, errorC chan Request, count int,
	reqUrl, key string, verbose bool, report *Report) {
	defer wg.Done()
	reqURL, _ := url.Parse(reqUrl)
	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"name":          fmt.Sprintf("request #%d", count),
		"date":          time.Now().String(),
		"requests_sent": count,
	})
	reqBody := bytes.NewReader(bodyBytes)
	body := ioutil.NopCloser(reqBody)
	req := &http.Request{
		Method: "POST",
		URL:    reqURL,
		Header: map[string][]string{
			"Content-Type": {"application/json; charset=UTF-8"},
			"X-Api-Key":    {key},
		},
		Body: body,
	}
	resp, err := http.DefaultClient.Do(req)
	report.TotalRequests += 1
	// properly handle http codes here
	// for instance, to retry a request or to collect the http status for error mapping
	if resp != nil {
		// valid success http status codes: 200, 201, 202, 204
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted &&
			resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
			report.TotalFail += 1
			errorC <- Request{
				Index:    count,
				Err:      err,
				Response: resp,
			}
		} else {
			report.TotalSuccess += 1
			resultC <- Request{
				Index:    count,
				Response: resp,
			}
		}
	}
	if err != nil {
		report.TotalFail += 1
		errorC <- Request{
			Index: count,
			Err:   err,
		}
	}
}

func processResults(bulk int, resultC chan Request, verbose bool) {
	count := 0
	for {
		select {
		case res := <-resultC:
			count++
			if count == 1 {
				log.Println("buffer #", bulk)
			}
			log.Printf("request #%d >> http status response %d", res.Index, res.Response.StatusCode)
			if verbose {
				respBody, err := ioutil.ReadAll(res.Response.Body)
				check(err)
				prettyResp, err := prettyPrint(respBody)
				check(err)
				log.Printf("request #%d >> response: %s", res.Index, string(prettyResp))
			}
			defer res.Response.Body.Close()
		default:
		}
	}
}

func processErrors(bulk int, errorC chan Request, verbose bool) {
	count := 0
	for {
		select {
		case err := <-errorC:
			count++
			if count == 1 {
				log.Println("buffer #", bulk)
			}
			if err.Err != nil {
				log.Printf("error on request #%d >> %v", err.Index, err.Err)
			} else {
				log.Printf("error on request #%d >> http status code: %d", err.Index, err.Response.StatusCode)
				if verbose {
					respBody, newErr := ioutil.ReadAll(err.Response.Body)
					check(newErr)
					if len(respBody) > 0 {
						log.Printf("request #%d >> response: %s", err.Index, string(respBody))
					}
				}
			}
		default:
		}
	}
}

func executeRequestWithTimer(url, key string, rqs int, verbose bool, report *Report) {
	count := 0
	for range time.Tick(time.Second * time.Duration(1)) {
		count++
		resultChannel := make(chan Request)
		errorChannel := make(chan Request)

		var wg sync.WaitGroup

		for idx := 1; idx <= rqs; idx++ {
			wg.Add(1)
			go doRequest(&wg, resultChannel, errorChannel, idx, url, key, verbose, report)
		}

		go processResults(count, resultChannel, verbose)
		go processErrors(count, errorChannel, verbose)
		wg.Wait()
	}
}

func executeRequests(url, key string, rqs, duration int, verbose bool) {
	report := Report{}
	log.Println("Waiting for all requests to be executed...")
	go executeRequestWithTimer(url, key, rqs, verbose, &report)
	time.Sleep(time.Second * time.Duration(duration+1))
	log.Println("Requests executed successfully.")
	log.Println("--------------------REPORT--------------------")
	jsonReport, err := json.Marshal(report)
	check(err)
	prettyReport, err := prettyPrint(jsonReport)
	check(err)
	fmt.Println(string(prettyReport))
}

func prettyPrint(b []byte) ([]byte, error) {
	var out bytes.Buffer
	err := json.Indent(&out, b, "", "  ")
	return out.Bytes(), err
}

func getFlags() (urlPtr, keyPtr string, rqsPtr, durationPtr int, verbosePtr bool) {
	flag.StringVar(&urlPtr, "url", "https://postman-echo.com/post", "the server POST url")
	flag.StringVar(&keyPtr, "key", "RIqhxTAKNGaSw2waOY2CW3LhLny2EpI27i56VA6N", "the server API key")
	flag.IntVar(&rqsPtr, "rqs", 10, "requests per seconds")
	flag.IntVar(&durationPtr, "duration", 1, "duration in seconds")
	flag.BoolVar(&verbosePtr, "verbose", false, "whether to print out the response of each request or not")
	flag.Parse()
	fmt.Println("url:", urlPtr)
	fmt.Println("key:", keyPtr)
	fmt.Println("rqs:", rqsPtr)
	fmt.Println("duration:", durationPtr)
	fmt.Println("verbose:", verbosePtr)
	return
}

func main() {
	flag.Usage = func() {
		fmt.Print("Options:\n")
		flag.PrintDefaults()
	}

	url, key, rqs, duration, verbose := getFlags()

	executeRequests(url, key, rqs, duration, verbose)
}
