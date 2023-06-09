package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
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

const UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36"

func apartment(url string) (apartment Apartment) {
	c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", UserAgent)
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

	c.Visit(url)
	return apartment
}

func linksToApartments(url string) (links []string) {
	c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", UserAgent)
		fmt.Println("\nPage", r.URL)
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Error", err.Error())
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

func lastPage(url string) (last int) {
	c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", UserAgent)
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Error", err.Error())
	})

	c.OnHTML("span.last a", func(h *colly.HTMLElement) {
		rel := h.Attr("href")
		root := "https://boligzonen.dk/"
		link := root + rel
		re := regexp.MustCompile(`-?\d+`)
		last, _ = strconv.Atoi(re.FindString(link))
		fmt.Println("Last", last)
	})

	c.Visit(url)
	return last
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
	// url for desired search (Rentals in Copenhagen), number of last page
	url := "https://boligzonen.dk/lejebolig/kobenhavn-kommune"
	last := lastPage(url)

	// create file and csv writer, add header
	file, _ := os.Create("records.csv")
	header := []string{"id", "rooms", "area", "rent", "latitude", "longitude"}
	w := csv.NewWriter(file)
	w.Write(header)

	// channel with capacity of a single page, waitgroups for synchronisation
	channel := make(chan Apartment)
	var scrapers sync.WaitGroup
	var writer sync.WaitGroup

	// launch csv writer goroutine
	writer.Add(1)
	go func() {
		defer writer.Done()

		for a := range channel {
			write(a, w)
		}
	}()

	// crawl pages
	for page := 1; page <= last; page++ {
		links := linksToApartments(url + "?page=" + strconv.Itoa(page))

		// launch scraper goroutines
		for _, link := range links {
			scrapers.Add(1)
			go func(link string, channel chan Apartment) {
				defer scrapers.Done()
				channel <- apartment(link)
			}(link, channel)
		}
	}

	// wait for all apartments to be sent through channel, close it, wait for writer goroutine
	scrapers.Wait()
	close(channel)
	writer.Wait()

	// flush csv writer and close file
	w.Flush()
	file.Close()
}
