package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

type NameBody struct {
	Name string
	Body []byte
}

func main() {
	doc := URLToImageList(os.Args[1])
	urls := docToImageList(doc)

	log.Printf("Successfully parsed files: \n %s \n", strings.Join(urls[:], "\n"))

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	var wg sync.WaitGroup
	wg.Add(len(urls))

	nameBodys := make([]NameBody, 0)
	for i, url := range urls {
		go func(i int, url string) {
			log.Printf("Start downloading from url: %s \n", url)
			imgBody := toImgBytes(url)
			fileName := strconv.Itoa(i) + "." + getExtension(url)
			nameBodys = append(
				nameBodys,
				NameBody{
					Name: fileName,
					Body: imgBody,
				},
			)

			wg.Done()
		}(i, url)
	}
	wg.Wait()

	for _, nameBody := range nameBodys {
		log.Printf("Start compressing to file: %s \n", nameBody.Name)
		f, err := w.Create(nameBody.Name)
		check(err)
		_, err = f.Write(nameBody.Body)
		check(err)
	}

	err := w.Close()
	check(err)

	outFile := strings.Join(strings.Fields(doc.Find("title").First().Text()), "_") + ".cbz"
	log.Printf("Succefully compress to CBZ, start upload to dropbox file: %s", outFile)
	resp := uploadToDropBox(outFile, buf)

	respBody, err := ioutil.ReadAll(resp.Body)
	check(err)
	log.Println(string(respBody))
}

func uploadToDropBox(outFile string, buf *bytes.Buffer) *http.Response {
	req, err := http.NewRequest(
		"POST",
		"https://content.dropboxapi.com/2/files/upload",
		buf,
	)
	check(err)
	req.Header.Add("Content-Type", "application/octet-stream")
	req.Header.Add("Authorization", "Bearer rTmQsGICWJAAAAAAAAABGKFelymqKFQNRvqDgh4IrCy9FjadTf6kCZi3XomLjkbR")
	req.Header.Add("Dropbox-API-Arg", fmt.Sprintf("{\"path\": \"/%s\",\"mode\": \"add\",\"autorename\": true,\"mute\": false}", outFile))
	resp, err := http.DefaultClient.Do(req)
	check(err)
	return resp
}

func getExtension(url string) string {
	index := strings.LastIndex(url, ".")
	if index >= 0 {
		return url[index+1:]
	}
	return ""
}

func toImgBytes(url string) []byte {
	resp, err := http.Get(url)
	check(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	check(err)
	return body
}

// URLToImageList parse imgs in url to a list of image urls
func URLToImageList(url string) *goquery.Document {
	resp, err := http.Get(url)
	check(err)
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	check(err)

	return doc
}

func docToImageList(doc *goquery.Document) []string {
	imageUrls := make([]string, 0)
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title
		value, _ := s.Attr("src")
		w, hasWidth := s.Attr("width")
		h, hasHeight := s.Attr("height")
		width := 0
		height := 0
		if hasWidth {
			width, _ = strconv.Atoi(w)
		}

		if hasHeight {
			height, _ = strconv.Atoi(h)
		}
		if width > 200 && height > 200 {
			imageUrls = append(imageUrls, value)
		}
	})

	return imageUrls
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
