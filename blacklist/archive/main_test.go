package main

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func encodeURIComponent1(str string) string {
	r := url.QueryEscape(str)
	r = strings.Replace(r, "+", "%20", -1)
	return r
}

func TestUrl(t *testing.T) {
	url := "https://tieba.baidu.com/f?kz=4421231059&mo_device=1&ssid=0&from=844b&uid=0&pu=usm@2,sz@320_1001,ta@iphone_2_7.0_24_67.0&bd_page_type=1&baiduid=16C9EAE1D7D54FAC1A178F5882C9EBD3&tj=h5_mobile_1_0_10_l4&referer=m.baidu.com?pn=0&"
	url = encodeURIComponent1(url)
	req, _ := http.Get("https://archive.org/wayback/available?url=" + url)
	defer req.Body.Close()
	body, _ := ioutil.ReadAll(req.Body)
	t.Log("body", string(body))
}
