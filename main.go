package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
)

type Apartment struct {
	Ref       int
	Rooms     int
	Area      int
	Rent      int
	Latitude  float64
	Longitude float64
}

func apartment(url string) (apartment Apartment) {
	c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36")
		fmt.Println("Visiting", r.URL)
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Error", err.Error())
	})

	c.OnHTML(".reference-number", func(h *colly.HTMLElement) {
		value := h.Text
		value = strings.Trim(value, "Sagsnummer: ")
		ref, _ := strconv.Atoi(value)
		apartment.Ref = ref
	})

	c.OnHTML(".section-bar", func(h *colly.HTMLElement) {
		selection := h.DOM
		label := selection.Find(".section-bar-label").Text()
		value := selection.Find(".section-bar-value").Text()

		switch label {
		case "Antal værelser":
			rooms, _ := strconv.Atoi(value)
			apartment.Rooms = rooms
		case "Størrelse":
			value = value[:len(value)-4]
			area, _ := strconv.Atoi(value)
			apartment.Area = area
		case "Husleje":
			value = strings.Trim(value, ",-")
			value = strings.ReplaceAll(value, ".", "")
			rent, _ := strconv.Atoi(value)
			apartment.Rent = rent
		}
	})

	c.OnHTML("div[data-lat][data-lng]", func(h *colly.HTMLElement) {
		lat := h.Attr("data-lat")
		lng := h.Attr("data-lng")
		latitude, _ := strconv.ParseFloat(lat, 64)
		longitude, _ := strconv.ParseFloat(lng, 64)
		apartment.Latitude = latitude
		apartment.Longitude = longitude

	})

	// c.OnScraped(func(r *colly.Response) {
	// 	enc := json.NewEncoder(os.Stdout)
	// 	enc.SetIndent("", " ")
	// 	enc.Encode(apartment)
	// })

	c.Visit(url)

	return apartment
}

func links(url string) (links []string) {
	c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36")
		fmt.Println("Visiting", r.URL)
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Error", err.Error())
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Println("Response Code", r.StatusCode)
	})

	c.OnHTML(".property-partial[href]", func(h *colly.HTMLElement) {
		rel := h.Attr("href")
		root := "https://boligzonen.dk/"
		link := root + rel
		links = append(links, link)
	})

	c.Visit(url)

	return links
}

func next(url string) (link string) {
	c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36")
		fmt.Println("Visiting", r.URL)
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Error", err.Error())
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Println("Response Code", r.StatusCode)
	})

	c.OnHTML("span.next a", func(h *colly.HTMLElement) {
		rel := h.Attr("href")
		root := "https://boligzonen.dk/"
		link = root + rel
		// fmt.Println(link)
	})

	c.Visit(url)
	return link
}

func last(url string) (link string) {
	c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36")
		fmt.Println("Visiting", r.URL)
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Error", err.Error())
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Println("Response Code", r.StatusCode)
	})

	c.OnHTML("span.last a", func(h *colly.HTMLElement) {
		rel := h.Attr("href")
		root := "https://boligzonen.dk/"
		link = root + rel
		// fmt.Println(link)
	})

	c.Visit(url)
	return link
}

func write(a Apartment, w *csv.Writer) {
	row := []string{
		fmt.Sprintf("%d", a.Ref),
		fmt.Sprintf("%d", a.Rooms),
		fmt.Sprintf("%d", a.Area),
		fmt.Sprintf("%d", a.Rent),
		fmt.Sprintf("%f", a.Latitude),
		fmt.Sprintf("%f", a.Longitude),
	}
	w.Write(row)
}

func main() {
	start := "https://boligzonen.dk/lejebolig/kobenhavn-kommune"
	last := last("https://boligzonen.dk/lejebolig/kobenhavn-kommune")
	// links := links("https://boligzonen.dk/lejebolig/kobenhavn-kommune")
	channel := make(chan Apartment)
	file, _ := os.Create("records.csv")
	w := csv.NewWriter(file)

	// synchronisation
	var scrapers sync.WaitGroup
	var writer sync.WaitGroup

	// csv writer
	writer.Add(1)
	go func() {
		defer writer.Done()

		for a := range channel {
			write(a, w)
		}
	}()

	// crawl pages
	for url := start; url != last; url = next(url) {
		links := links(url)

		// launch scraper goroutines
		for _, link := range links {
			scrapers.Add(1)
			go func(link string, channel chan Apartment) {
				defer scrapers.Done()

				channel <- apartment(link)
			}(link, channel)
		}
	}

	// synchronisation
	scrapers.Wait()
	close(channel)
	writer.Wait()

	// flush writer and close file
	w.Flush()
	file.Close()
}
