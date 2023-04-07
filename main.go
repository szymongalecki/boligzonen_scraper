package main

import (
	"fmt"
	"strconv"
	"strings"

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
			value = strings.Trim(value, "m2")
			value = strings.TrimSpace(value)
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

func main() {
	links := links("https://boligzonen.dk/lejebolig/kobenhavn-kommune")
	apartments := make(chan Apartment)

	for _, link := range links {
		go func(link string, apartments chan Apartment) {
			apartments <- apartment(link)
		}(link, apartments)
	}

	for i := 0; i < 18; i++ {
		fmt.Println(<-apartments)
	}
}
