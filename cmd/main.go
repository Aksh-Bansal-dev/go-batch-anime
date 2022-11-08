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
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gocolly/colly"
	"github.com/joho/godotenv"
)

var (
	animeName  = flag.String("n", "", "name of anime")
	startEp    = flag.Int("start", 1, "starting episode")
	endEp      = flag.Int("end", 0, "ending episode")
	resolution = flag.String("res", "1280", "resolution")
)

// WriteCounter counts the number of bytes written to it. It implements to the io.Writer interface
// and we can pass this into io.TeeReader() which will report progress on each write cycle.
type WriteCounter struct {
	Total      uint64
	checkTime  time.Time
	checkTotal uint64
	bandwidth  uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	if time.Now().Sub(wc.checkTime) > time.Second {
		wc.bandwidth = (wc.Total - wc.checkTotal) / 1024
		wc.checkTime = time.Now()
		wc.checkTotal = wc.Total
	}
	wc.PrintProgress()
	return n, nil
}

func (wc WriteCounter) PrintProgress() {
	// Clear the line by using a character return to go back to the start and remove
	// the remaining characters by filling it with spaces
	fmt.Printf("\r%s", strings.Repeat(" ", 45))
	// Return again and print current status of download
	// We use the humanize package to print the bytes in a meaningful way (e.g. 10 MB)
	fmt.Printf("\rDownloading... %s complete [%d KB/s]", humanize.Bytes(wc.Total), wc.bandwidth)
}

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
			// resp, err := http.Get(e.Attr("href"))
			// if err != nil {
			// 	log.Fatal(err)
			// }
			// defer resp.Body.Close()
			// out, err := os.Create(downloadPath)
			// defer out.Close()
			// if err != nil {
			// 	log.Fatal("file error", err)
			// }
			// fmt.Println("Downloading at", downloadPath)
			// _, err = io.Copy(out, resp.Body)

			err := DownloadFile(homeDir, e.Attr("href"))
			if err != nil {
				log.Fatal(err)
			}
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

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory. We pass an io.TeeReader
// into Copy() to report progress on the download.
func DownloadFile(homeDir string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	filepath := path.Join(homeDir, "Downloads", *animeName, resp.Request.URL.Query().Get("title"))
	fmt.Println("Downloading at", filepath)
	// Create the file, but give it a tmp file extension, this means we won't overwrite a
	// file until it's downloaded, but we'll remove the tmp extension once downloaded.
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err
	}

	// Create our progress reporter and pass it to be used alongside our writer
	counter := &WriteCounter{}
	if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		out.Close()
		return err
	}

	// The progress use the same line so print a new line once it's finished downloading
	fmt.Print("\n")

	// Close the file without defer so it can happen before Rename()
	out.Close()

	if err = os.Rename(filepath+".tmp", filepath); err != nil {
		return err
	}
	return nil
}
