package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"
)

func main() {
	cwd, _ := os.Getwd()
	filePath := "/blacklist/README.md"
	filePath = path.Join(cwd, filePath)
	log.Println("cwd", cwd, filePath)
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal(err)
	}
	content := string(data)
	reg := regexp.MustCompile(`\(https?://.*?\)`)
	strArr := reg.FindAllString(content, -1)
	log.Printf("%q\n", strArr)
	chanlen := 10
	check := make(chan string, chanlen)
	var wg sync.WaitGroup
	for i := 0; i < chanlen; i++ {
		wg.Add(1)
		go processUrl(check, &wg)
	}
	for _, str := range strArr {
		str = str[1 : len(str)-1]
		log.Println("url", str)
		check <- str
	}
	close(check)
	wg.Wait()
	log.Println("all end", len(strArr))
}

func processUrl(check chan string, wg *sync.WaitGroup) {

	defer wg.Done()
	client := &http.Client{}
	for {
		str, ok := <-check
		if !ok {
			log.Println("process end")
			return
		}
		u := encodeURIComponent(str)
		b, err := checkExists(client, u)
		if err != nil {
			log.Println("url error:", str, b, err)
		}

		if !b {
			log.Println("url save:", str)
			err = saveUrl(client, str)
			if err != nil {
				log.Println("save url error", str, err)
			}
		}
	}

}

type availableResponse struct {
	ArchivedSnapshots struct {
		Closest struct {
			Available bool   `json:"available"`
			Status    string `json:"status"`
			Timestamp string `json:"timestamp"`
			URL       string `json:"url"`
		} `json:"closest"`
	} `json:"archived_snapshots"`
	URL string `json:"url"`
}

func checkExists(client *http.Client, url string) (bool, error) {
	req, err := http.NewRequest("GET", "https://archive.org/wayback/available?url="+strings.Split(url, "?")[0], nil)
	if err != nil {
		return false, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 503 {
		// 503 Service Temporarily Unavailable
		time.Sleep(time.Second * 2)
		return checkExists(client, url)
	}
	body, err := ioutil.ReadAll(resp.Body)
	var r availableResponse
	err = json.Unmarshal(body, &r)
	if err != nil {
		log.Println("err body", string(body))
		return false, err
	}
	log.Printf("url:%s timestamp:%s\n", url, r.ArchivedSnapshots.Closest.Timestamp)
	return r.ArchivedSnapshots.Closest.Timestamp != "", nil
}

func saveUrl(client *http.Client, u string) error {
	req, err := http.NewRequest("GET", "https://web.archive.org/save/"+strings.Split(u, "?")[0], nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func encodeURIComponent(str string) string {
	r := url.QueryEscape(str)
	r = strings.Replace(r, "+", "%20", -1)
	return r
}
