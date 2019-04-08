package main

import (
	"context"
	"encoding/json"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"testing"
)

func TestGetProfile(t *testing.T) {
	t.Log("test begin")
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: TOKEN},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	chm, resp, err := client.Repositories.GetCommunityHealthMetrics(context.TODO(), "996icu", "996.ICU")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if chm == nil || resp.StatusCode != 200 {
		t.Fatal("resp code error", resp)
	}
	jv, _ := json.Marshal(chm)
	t.Logf("%s", string(jv))
}

func TestGetReadme(t *testing.T) {
	t.Log("test begin")
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: TOKEN},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	chm, resp, err := client.Repositories.GetReadme(context.TODO(), "BAHome", "BAButton", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if chm == nil || resp.StatusCode != 200 {
		t.Fatal("resp code error", resp)
	}
	jv, _ := json.Marshal(chm)
	t.Logf("readme:%s", string(jv))

	gr, resp, err := client.Repositories.License(context.TODO(), "corpnewt", "gibMacOS")
	jv, _ = json.Marshal(gr)
	t.Logf("license:%s", string(jv))
}
