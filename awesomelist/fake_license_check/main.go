package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"
)

var fakeRepository = make(chan string, 20)

const githubPrefix = "(https://github.com/"
const githubPrefixLen = len(githubPrefix)
const projectPath = "/awesomelist/projects.md"

var TOKEN = ""

func init() {
	TOKEN = os.Getenv("GH_TOKEN")
	if TOKEN == "" {
		cwd, _ := os.Getwd()
		if !strings.HasSuffix(cwd, "fake_license_check") {
			cwd += "/awesomelist/fake_license_check/.token"
		} else {
			cwd += "/.token"
		}
		data, err := ioutil.ReadFile(cwd)
		if err != nil {
			log.Fatal("failed get github token", cwd)
		}
		TOKEN = string(data)
	}
}
func main() {
	cwd, _ := os.Getwd()
	filePath := projectPath
	filePath = path.Join(cwd, filePath)
	log.Println("cwd", cwd, filePath)
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal(err)
	}
	content := string(data)
	reg := regexp.MustCompile(`\(https://github.com/.*?\)`)
	strArr := reg.FindAllString(content, -1)
	strArr = RemoveDuplicatesAndEmpty(strArr)
	urls, _ := json.Marshal(strArr)
	log.Printf("%s\n", urls)

	chanlen := 10
	check := make(chan string, chanlen)
	var wg sync.WaitGroup
	processChan := make(chan struct{})
	go processFakeRepo(processChan, content)
	for i := 0; i < chanlen; i++ {
		wg.Add(1)
		go processUrl(check, &wg)
	}
	for idx, str := range strArr {
		str = str[githubPrefixLen : len(str)-1]
		if str[len(str)-1] == '/' {
			str = str[:len(str)-1]
		}
		if strings.Count(str, "/") == 0 {
			continue
		}
		log.Println(idx, str)

		check <- str
		time.Sleep(time.Second * 5)

	}
	close(check)
	wg.Wait()
	log.Println("check goroutines end")
	close(fakeRepository)
	<-processChan
	close(processChan)
	log.Println("check fake success")
}

func processFakeRepo(processChan chan struct{}, content string) {
	defer func() { processChan <- struct{}{} }()
	fakeNames := make([]string, 0, 10)
	scanner := bufio.NewScanner(strings.NewReader(content))
	linearr := make([]string, 0, len(content)/128)
	for scanner.Scan() {
		linearr = append(linearr, scanner.Text())
	}
	rmIdxs := make([]int, 0, 30)
	for repoName := range fakeRepository {
		log.Println("recv fakeRepo", repoName)
		for idx, str := range linearr {
			if strings.Index(str, repoName) != -1 {
				rmIdxs = append(rmIdxs, idx)
			}
		}
		fakeNames = append(fakeNames, repoName)
	}
	if len(fakeNames) == 0 {
		log.Println("processFakeRepo end without fake repo")
		return
	}
	log.Println("fake repos", fakeNames)
	// process fakeRepo

	for v, idx := range rmIdxs {
		linearr = append(linearr[:idx-v], linearr[idx-v+1:]...)
	}
	createPR(fakeNames, strings.Join(linearr, "\n"))
}

type PrBot struct {
	Commit      string `json:"commit"`
	Description string `json:"description"`
	Files       []struct {
		Content string `json:"content"`
		Path    string `json:"path"`
	} `json:"files"`
	Repo  string `json:"repo"`
	Title string `json:"title"`
	Token string `json:"token"`
	User  string `json:"user"`
}

func createPR(fakeRepo []string, content string) error {
	const url = "https://xrbhog4g8g.execute-api.eu-west-2.amazonaws.com/prod/prb0t"
	const contentType = "application/json; charset=utf-8"
	prbot := PrBot{}
	prbot.User = "whtiehack"
	prbot.Repo = "996.ICU"
	prbot.Description = "Fake 996 license repositories:\n" + githubPrefix[1:] + strings.Join(fakeRepo, "\n"+githubPrefix[1:]) + " "
	prbot.Title = "Remove fake 996 license repositories."
	prbot.Commit = prbot.Title
	prbot.Files = []struct {
		Content string `json:"content"`;
		Path    string `json:"path"`
	}{
		{Content: content, Path: projectPath[1:]},
	}
	prbot.Token = TOKEN
	pb, _ := json.Marshal(prbot)
	log.Println("pb", string(pb))
	log.Println("Description", prbot.Description)
	ioutil.WriteFile(".projects.md", []byte(content), 0777)

	//resp, err := http.Post(url, contentType, bytes.NewReader(pb))
	//if err != nil {
	//	log.Println("createPR error", err)
	//	return err
	//}
	//defer resp.Body.Close()
	//body, _ := ioutil.ReadAll(resp.Body)
	//log.Println("createPR result", string(body))
	return nil
}

func processUrl(c <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	count := 0
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: TOKEN},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	for repoName := range c {
		count++
		time.Sleep(time.Second * 10)
		log.Println("begin check ", repoName)
		b, err := CheckHas996(repoName, client)
		if err != nil {
			log.Println("error", repoName, err)
			continue
		}
		if !b {
			// 假996license仓库
			fakeRepository <- repoName
		}
	}
	log.Println("process end", count)
}

func checkContent(content string) bool {
	arr := strings.Split(content, "\n")
	for _, str := range arr {
		v, _ := base64.StdEncoding.DecodeString(str)
		if bytes.Index(v, []byte("996")) >= 0 {
			return true
		}
	}
	return false
}

func CheckHas996(repo string, client *github.Client) (bool, error) {
	arr := strings.Split(repo, "/")
	chm, resp, err := client.Repositories.GetReadme(context.TODO(), arr[0], arr[1], nil)
	if err != nil || chm == nil || chm.Content == nil {
		return false, err
	}
	defer resp.Body.Close()
	if checkContent(*chm.Content) {
		return true, nil
	}
	gr, respLicense, err := client.Repositories.License(context.TODO(), arr[0], arr[1], )
	if err != nil || gr == nil || gr.Content == nil {
		return false, err
	}
	defer respLicense.Body.Close()
	return checkContent(*gr.Content), nil
}

func checkHas996Newer(repo string, client *github.Client) (bool, error) {
	// https://developer.github.com/v3/repos/community/
	arr := strings.Split(repo, "/")
	if len(arr) != 2 {
		return false, errors.New("checkHas996Newer,repo error:" + repo)
	}
	chm, resp, err := client.Repositories.GetCommunityHealthMetrics(context.TODO(), arr[0], arr[2])
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if chm == nil || resp.StatusCode != 200 {
		return false, nil
	}
	return true, nil
}

func RemoveDuplicatesAndEmpty(a []string) (ret []string) {
	a_len := len(a)
	for i := 0; i < a_len; i++ {
		if (i > 0 && a[i-1] == a[i]) || len(a[i]) == 0 {
			continue;
		}
		ret = append(ret, a[i])
	}
	return
}
