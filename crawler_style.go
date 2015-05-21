package main

import (
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	//    "runtime"
	//    "github.com/jeffail/tunny"
)

const (
	SaveDir = "/tmp/style.com/"
)

func init() {
	//    numCPUs := runtime.NumCPU()
	//    runtime.GOMAXPROCS(numCPUs)
	//    pool, _ := tunny.CreatePoolGeneric(numCPUs).Open()
	//    defer pool.Close()
}

func main() {

	processSeasonUrl("http://www.style.com/fashion-shows/pre-fall-2015/")
}

func processSeasonUrl(url string) {
	doc, err := goquery.NewDocument(url)
	if err != nil {
		log.Fatal(err)
		return
	}
	brands := make([]string, 0, 0)
	doc.Find("#s0-all li a").Each(func(i int, s *goquery.Selection) {
		href, exist := s.Attr("href")
		if !exist {
			log.Fatal("not exist : ", href)
		}
		brand := href[strings.LastIndex(href, "/")+1 : len(href)]
		brands = append(brands, brand)
	})
	wg := sync.WaitGroup{}
	for i, b := range brands {
		wg.Add(1)

		wg2 := sync.WaitGroup{}
		wg2.Add(1)
		go processCollection(b, i, len(brands), &wg, &wg2)
		wg2.Wait()
	}

	wg.Wait()
}

func processCollection(brand string, index int, total int, wg *sync.WaitGroup, wg2 *sync.WaitGroup) {
	defer wg.Done()
	defer wg2.Done()
	info := "processing: brand " + brand + " " + strconv.Itoa(index) + "/" + strconv.Itoa(total)
	log.Println(info)

	saveTo := SaveDir + brand + "/"
	collectionUrl := "http://www.style.com/slideshows/fashion-shows/pre-fall-2015/" + brand + "/collection"

	if doc, err := goquery.NewDocument(collectionUrl); err != nil {
		log.Fatal(err)
	} else {
		doc.Find("script").Each(func(i int, s *goquery.Selection) {
			if i == 4 {
				scriptStr := s.Text()
				scriptStr = strings.Replace(scriptStr, "<script>", "", -1)
				scriptStr = strings.Replace(scriptStr, "window.slideshowItems =", "", -1)
				scriptStr = strings.Replace(scriptStr, ";\n</script>", "", -1)
				scriptStr = strings.Replace(scriptStr, "\"isStatic\":false};", "\"isStatic\":false}", -1)
				//                log.Println(scriptStr)
				b := []byte(scriptStr)

				resultMap := ResultMap{}
				if err := json.Unmarshal(b, &resultMap); err != nil {
					log.Fatal(err)
				} else {

					wg := sync.WaitGroup{}
					for idx, item := range resultMap.Items {
						url := "http://media.style.com/image" + item.Slidepath
						pos := strings.LastIndex(url, "collection/") + len("collection/")
						url = url[:pos] + "683/1024/" + url[pos:]
						//                        log.Println(url)
						saveName := url[strings.LastIndex(url, "/")+1:]

						itemInfo := info + ", image " + strconv.Itoa(idx) + "/" + strconv.Itoa(len(resultMap.Items))

						wg.Add(1)
						go saveImage(itemInfo, url, saveTo, saveName, &wg)

						if item.HasDetailSlides {
							wg := sync.WaitGroup{}
							for i, detail := range item.Details {
								detailUrl := "http://media.style.com/image" + detail.SlidePath
								pos := strings.LastIndex(detailUrl, "detail/") + len("detail/")
								detailUrl = detailUrl[:pos] + "683/1024/" + detailUrl[pos:]
								saveNameDetail := strings.Replace(saveName, ".jpg", "", -1) + "_" + strconv.Itoa(i) + ".jpg"

								detailInfo := info + ", detail " + strconv.Itoa(idx) + "/" + strconv.Itoa(len(resultMap.Items))
								wg.Add(1)
								go saveImage(detailInfo, detailUrl, saveTo, saveNameDetail, &wg)
							}
							wg.Wait()
						}
					}
					wg.Wait()
				}
			}
		})
	}
	log.Println(info + ", done")
}

func saveImage(description string, url string, savePath string, saveName string, wg *sync.WaitGroup) {
	defer wg.Done()

	log.Println(description)
	var resp *http.Response
	resp, err := http.Get(url)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	} else {
		log.Fatal(err)
	}
	if resp == nil || resp.Body == nil || err != nil || resp.StatusCode != http.StatusOK {
		log.Println("error : " + description)
		log.Println(err)
		return
	}
	var buf []byte
	buf, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		if err1 := os.MkdirAll(savePath, os.ModePerm); err1 != nil {
			log.Fatal("error creating directory " + savePath)
			log.Fatal(err1)
		} else {
			log.Println("mkdir : " + savePath)
		}
	}

	ioutil.WriteFile(savePath+saveName, buf, os.ModePerm)

	log.Println(description + ", done")
}

type ResultMap struct {
	//    Id string
	//    Title string
	//    slideCount int
	//    seasonUrlFragment string
	//    brandUrlFragment string
	//    canonicalUrl string
	Items []Item
}

type Item struct {
	//    Id string
	//    order int
	Slidepath       string
	HasDetailSlides bool
	//    height int
	//    width int
	Details []Detail
}

type Detail struct {
	//    id string
	//    order int
	SlidePath string
	//    height int
	//    width int
}
