package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gocolly/colly"
	"github.com/joho/godotenv"
)

var (
	animeName  = flag.String("n", "", "name of anime")
	startEp    = flag.Int("start", 1, "starting episode")
	endEp      = flag.Int("end", 0, "ending episode")
	resolution = flag.String("res", "1280", "resolution")
)

func main() {
	flag.Parse()
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	if *animeName == "" {
		log.Fatal("No anime name found")
	}
	if *endEp == 0 {
		log.Fatal("No end episode found")
	}
	var cookie = os.Getenv("AUTH_COOKIE")
	homeDir, err := os.UserHomeDir()
	downloadDir := path.Join(homeDir, "Downloads", *animeName)
	err = os.MkdirAll(downloadDir, os.ModePerm)
	if err != nil {
		log.Fatal("home dir error", err)
	}
	c := colly.NewCollector()

	c.OnHTML("div.cf-download a[href]", func(e *colly.HTMLElement) {
		if strings.Contains(e.Text, *resolution) {
			resp, err := http.Get(e.Attr("href"))
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()
			// fmt.Println("Downloading", resp.Request.URL.Query().Get("title"), "...")
			downloadPath := path.Join(homeDir, "Downloads", *animeName, resp.Request.URL.Query().Get("title"))
			out, err := os.Create(downloadPath)
			defer out.Close()
			if err != nil {
				log.Fatal("file error", err)
			}
			fmt.Println("Downloading at", downloadPath)
			_, err = io.Copy(out, resp.Body)
			fmt.Println("Complete!")
		}
	})

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("cookie", cookie)
		fmt.Println("Visiting", r.URL)
	})

	for i := *startEp; i <= *endEp; i++ {
		c.Visit(fmt.Sprintf("https://gogoanime.tel/%s-episode-%d", *animeName, i))
	}
}
