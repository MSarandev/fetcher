package main

import (
	"encoding/xml"
	"html/template"
	"io/ioutil"
	"net/http"
	"sync"
)

// define the waitgroup
var wg sync.WaitGroup
var mappedQueue = make(chan MappedData, 100)
var notMappedQueue = make(chan DataIdentifier, 100) // make a new channel
var linksMap = make(map[int]DataReturned)

// Struct to contain the returned data
type DataReturned struct {
	Url string
}

// Struct to define where the data lives
type DataIdentifier struct {
	Locations []string `xml:"sitemap>loc"` // a slice of locations (where the url is)
} 

// Struct for mapped data
type MappedData struct{
	Url []string `xml:"url>loc"`
}

// Function to retrieve the data
func retrieveFromUrl(urlString string, c chan DataIdentifier){
	defer wg.Done()

	var dc DataIdentifier

	resp, _ := http.Get(urlString)
	bytes, _ := ioutil.ReadAll(resp.Body) // fetch the body
	resp.Body.Close()                     // close request (free resources)

	xml.Unmarshal(bytes, &dc) // unmarshal the data from the request

	c <- dc
}

// Function to map the data
func mapDataFromUrl(loc string, c chan MappedData){
	defer wg.Done()

	var mc MappedData

	resp, _ := http.Get(loc)
	bytes, _ := ioutil.ReadAll(resp.Body) // fetch the body
	resp.Body.Close()                     // close request (free resources)

	xml.Unmarshal(bytes, &mc) // unmarshal the data from the request

	c <- mc
}

// define the page
type LinksPage struct {
	Title string
	Links  map[int]DataReturned
}

// web data serve function
func serveToWeb(w http.ResponseWriter, r *http.Request){
	// define the page, and parse data
	p := LinksPage{Title: "Aggregator 3000", Links: linksMap}

	t, _ := template.ParseFiles("base-template.html")

	// execute the page
	t.Execute(w, p)
}

func main(){
	wg.Add(1) // begin the wrapper
	go retrieveFromUrl("https://www.uber.com/sitemap.xml", notMappedQueue)

	wg.Wait()
	close(notMappedQueue)

	//foreach top-location, map the sub-locations
	for el := range notMappedQueue{
		for idx := range el.Locations { // foreach news title
			wg.Add(1)
			//go mapDataFromUrl(el.Locations[idx], mappedQueue)

			// make the map
			linksMap[idx] = DataReturned{
				Url: el.Locations[idx],
			}
		}
	}

	close(mappedQueue)

	// Add the webserver
	http.HandleFunc("/", serveToWeb)
	http.ListenAndServe(":8181", nil)
}