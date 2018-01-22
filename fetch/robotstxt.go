package fetch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/temoto/robotstxt"
)

//IsRobotsTxt returns true if resource is robots.txt file
func IsRobotsTxt(url string) bool {
	return strings.HasSuffix(url, "/robots.txt")
}

//fetchRobots is used for getting robots.txt files.
func fetchRobots(req BaseFetcherRequest) (*BaseFetcherResponse, error) {
	//fetch content
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(b)

	//https://gitlab.com/gitlab-org/gitlab-ce/issues/33534
	//Use BaseFetcher to get robots.txt content
	addr := "http://" + viper.GetString("DFK_FETCH") + "/response/base"
	request, err := http.NewRequest("POST", addr, reader)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	r, err := client.Do(request)
	//if r != nil {
	//	defer r.Body.Close()
	//}
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	bfResponse := &BaseFetcherResponse{}

	if err := json.Unmarshal(data, &bfResponse); err != nil {
		return nil, err
	}
	return bfResponse, nil
}



//RobotstxtData generates robots.txt url, retrieves its content through API fetch endpoint.
func RobotstxtData(url string) (robotsData *robotstxt.RobotsData, err error) {
	parsedURL, err := neturl.Parse(url)
	if err != nil {
		return nil, err
	}
	//generate robotsURL from req.URL
	robotsURL := fmt.Sprintf("%s://%s/robots.txt", parsedURL.Scheme, parsedURL.Host)
	r := BaseFetcherRequest{URL: robotsURL, Method: "GET"}

	//response, err := fetchRobots(r)
	response, err := fetchRobots(r)

	if err != nil {
		return nil, err
	}
	//FromStatusAndBytes takes into consideration returned statuses along with robots.txt content
	// From https://developers.google.com/webmasters/control-crawl-index/docs/robots_txt
	//
	// Google treats all 4xx errors in the same way and assumes that no valid
	// robots.txt file exists. It is assumed that there are no restrictions.
	// This is a "full allow" for crawling. Note: this includes 401
	// "Unauthorized" and 403 "Forbidden" HTTP result codes.
	//
	// From Google's spec:
	// Server errors (5xx) are seen as temporary errors that result in a "full
	// disallow" of crawling.
	robotsData, err = robotstxt.FromStatusAndBytes(response.StatusCode, response.HTML)
	return
}


//AllowedByRobots checks if scraping of specified URL is allowed by robots.txt
func AllowedByRobots(url string, robotsData *robotstxt.RobotsData) bool {
	if robotsData == nil {
		return true
	}
	parsedURL, err := neturl.Parse(url)
	if err != nil {
		logger.Error("err")
	}
	return robotsData.TestAgent(parsedURL.Path, "DataflowKitBot")
}

//GetCrawlDelay retrieves Crawl-delay directive from robots.txt. Crawl-delay is not in the standard robots.txt protocol, and according to Wikipedia, some bots have different interpretations for this value. That's why maybe many websites don't even bother defining the rate limits in robots.txt. Crawl-delay value does not have an effect on delays between consecutive requests to the same domain for the moment. FetchDelay and RandomizeFetchDelay from ScrapeOptions are used for throttling a crawler speed.
func GetCrawlDelay(r *robotstxt.RobotsData) time.Duration {
	if r != nil {
		group := r.FindGroup("DataflowKitBot")
		return group.CrawlDelay
	}
	return 0
}
