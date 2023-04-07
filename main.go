package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
)

type Apartment struct {
	Rooms int
	Area  int
	Rent  int
}

func main() {
	apartment := Apartment{}
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

	c.OnHTML(".section-bar", func(h *colly.HTMLElement) {
		selection := h.DOM

		label := selection.Find(".section-bar-label").Text()
		value := selection.Find(".section-bar-value").Text()

		switch label {
		case "Antal værelser":
			rooms, _ := strconv.Atoi(value)
			// fmt.Println("Rooms", rooms)
			apartment.Rooms = rooms
		case "Størrelse":
			value = strings.Trim(value, "m2")
			value = strings.TrimSpace(value)
			area, _ := strconv.Atoi(value)
			// fmt.Println("Area", area)
			apartment.Area = area
		case "Husleje":
			value = strings.Trim(value, ",-")
			value = strings.ReplaceAll(value, ".", "")
			rent, _ := strconv.Atoi(value)
			// fmt.Println("Rent", rent)
			apartment.Rent = rent
		}
	})

	c.OnScraped(func(r *colly.Response) {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", " ")
		enc.Encode(apartment)
	})

	c.Visit("https://boligzonen.dk/lejeboliger/3-vaerelses-lejlighed-i-kastrup-10")
}
