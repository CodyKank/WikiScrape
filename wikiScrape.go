package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/html"
)

func getFullLinks(wikiToken *html.Tokenizer) ([]string, int) {
	//Function which will take in a *html.tokenizer and find all of the full
	//links (/wiki/foo/bar.php.meow) and return them as a splice of strings.

	// Initially creating an array of 520 to hold the links
	var tempLinks [1500]string
	var numberLinks int = 0

	for {
		nextToken := wikiToken.Next()

		switch {
		case nextToken == html.ErrorToken:
			// This would be the end of the HTML document, so exit!
			return tempLinks[0:numberLinks], numberLinks

		case nextToken == html.StartTagToken:
			t := wikiToken.Token()

			linkBool := t.Data == "a"
			if linkBool {
				for _, a := range t.Attr {
					// Have to check all attributes belonging to token t
					if a.Key == "href" {
						tempLinks[numberLinks] = a.Val // appending a link to the array
						numberLinks++
						// If we found the link, we don't want to check the rest of the
						// attributes!
						break
					}
				}
			}

		}
	}
	return tempLinks[0:(numberLinks + 1)], numberLinks
}

func getImageLinks(fullLinks []string) ([]string, int) {
	//Function to parse through all of the links found on the wikipage for only
	//links which are images. This is due to how mediawiki stores images.

	var images [800]string
	var count int = 0

	for _, link := range fullLinks {
		if strings.HasPrefix(link, "/wiki/images/") {
			images[count] = link
			count++
		}
	}
	return images[0:count], count
}

func downloadImages(imageLinks []string, urlPrefix string) bool {
	//Function to download all of the images within the imageLinks slice. first
	//function will create a directory called WikiImages. There, it will put all
	//of the images it downloads.

	saveDir := "WikiImages"

	//Creating a directory for all of the images to be placed into
	if _, err := os.Stat(saveDir); os.IsNotExist(err) {
		os.Mkdir(saveDir, 0777)
	}

	//for each image, download it
	for _, image := range imageLinks {
		//need to get the /wiki/images/0/0/ out of the name!!!
		imageName := strings.Replace(image, "/wiki/images/", "", -1)
		name := fmt.Sprintf("%s%s", saveDir, imageName[4:])
		fullURL := fmt.Sprintf("%s%s", urlPrefix, image)

		fmt.Println(fullURL, "  ", name)
		//Begin process of downloading the actual file
		out, err := os.Create(name)
		if err != nil {
			fmt.Println("Error: Could not create image,", name)
			fmt.Println("Please ensure the wikiImages dir does not exist prior to executing program.")
			panic(err)
		}

		defer out.Close()

		resp, err := http.Get(fullURL)
		if err != nil {
			fmt.Println("Error: Could not download file located at: ", fullURL)
			panic(err)
		}
		defer resp.Body.Close()

		_, errorCheck := io.Copy(out, resp.Body)
		if errorCheck != nil {
			if os.IsExist(errorCheck) {
				printVar := fmt.Sprintf("File: %s already exists!", image)
				fmt.Println(printVar)
			} else {
				fmt.Println("Error: Could not create file for: ", name)
				panic(errorCheck)
			}
		}
	}
	return true
}

func getTitles(links []string) []string {
	// Function which will parse the wiki links for just the titles. Will only
	// look at links which qualify: /wiki/index.php/.* in regexp terms.
	var names [520]string
	var titles [520]string
	//Golang is functional so that means all values are immutable, need a copy!
	var numTitles int = 0

	for _, link := range links {
		if strings.HasPrefix(link, "/wiki/index.php/") {
			names[numTitles] = link
			numTitles++
		}
	}

	i := 0
	for _, title := range names {
		titles[i] = strings.Replace(title, "/wiki/index.php/", "", -1)
		i++
	}

	return titles[6:numTitles]
}

func grabURLPrefix(myURL string) string {
	//Function to find the urlprefix of any given wiki url (or any url). Will first
	//ignore the http:// etc prefix, then will find the first instance of '/'
	//and reap what it finds before that first '/', thus finding the url prefix.
	//This is necesarry as all of the links within a wiki do not have the prefix.

	var scrapedURL string
	var myProtocol string
	if strings.HasPrefix(myURL, "http://") {
		scrapedURL = myURL[7:]
		myProtocol = "http://"
	} else if strings.HasPrefix(myURL, "https://") {
		scrapedURL = myURL[8:]
		myProtocol = "https://"
	} else {
		fmt.Println("Cannot find url protocol. . . Winging it! Expect errors.")
		scrapedURL = myURL
	}
	var prefixIndex int = strings.Index(scrapedURL, "/")
	var urlPrefix string = scrapedURL[:prefixIndex]
	var buffer bytes.Buffer
	buffer.WriteString(myProtocol)
	buffer.WriteString(urlPrefix)
	return buffer.String()
}

func imageMain(imageURL string) {
	//Function to act as driver for finding and downloading all of the images found
	//within a mediawiki. Must be given the proper URL. See README for more information.

	response, err := http.Get(imageURL)
	if err != nil {
		panic(err)
	}

	token := html.NewTokenizer(response.Body)
	fullLinks, numLinks := getFullLinks(token)
	if numLinks == 0 {
		panic("Error: no links!")
	}

	imageLinks, numImages := getImageLinks(fullLinks)

	if numImages == 0 {
		panic("Error: No images were returned. Check wiki Link.")
	}

	urlPrefix := grabURLPrefix(imageURL)
	//error checking is built into the function, if something fails prog will panic
	downloadImages(imageLinks, urlPrefix)
	return
}

func writeTitles(titles []string) bool {
	//Function to write all of the wiki titles to a file. This will be a text file
	//which will then hold all of the titles to the wiki submitted. The file will
	//have UNIX permissions of 666. Returns a true if a file is created, false otherwise.

	titleFile, err := os.Create("WikiTitles.txt")
	if err != nil {
		if os.IsExist(err) {
			cmdReader := bufio.NewReader(os.Stdin)
			fmt.Println("Error: wikiTitles.txt already exists. Remove? [Y/N]:")
			answer, more, err := cmdReader.ReadLine()
			if more {
				fmt.Println("Please only answer on 1 line. Reading only 1 line.")
			}
			if err != nil {
				fmt.Println("Error: Cannot read Stdin.")
				panic(err)
			}
			switch string(answer) {
			case "Y", "y", "yes", "YES", "Yes":
				os.Remove("wikiTitles.txt")
				fmt.Println("Removed wikiTitles.txt. Moving on...")
				titleFile, _ = os.Create("WikiTitles.txt")
			case "N", "No", "no", "NO":
				fmt.Println("Exiting operations...")
				return false
			}
		} else {
			panic(err)
		}
	}
	//Making sure the file will close after we are done writing to it
	defer titleFile.Close()

	for _, title := range titles {
		titleFile.WriteString(title + "\n")
	}
	titleFile.Sync()
	return true
}

func titleMain(titleLink string) {
	//Funciton that will take in a string to represent the URL of the Wiki's Special
	//pages link which is intended to be wiki/foo/bar/title=Special:AncientPages with
	//the limit set to 500. This will only work with wiki's with less than 500 pages.

	response, err := http.Get(titleLink)
	if err != nil {
		fmt.Println("Error: Could not follow link given for titles. Consult the README for more info.\n")
		panic(err)
	}

	token := html.NewTokenizer(response.Body)
	fullLinks, numLinks := getFullLinks(token)

	if numLinks == 0 { //error checking
		panic("Error, number of links is 0! Check the link submitted against info in README.")
	}

	titles := getTitles(fullLinks)

	checking := writeTitles(titles)
	if !checking {
		fmt.Println("Did not write wiki's titles to file.")
	} else {
		fmt.Println("Wrote all titles to wikiTitles.txt")
	}
	return
}

func show_usage(code int) {
	//Function to demonstrate the usage. Will die after showing usage.
	fmt.Println("Usage:")
	fmt.Printf("    %s -images \"[URL for file list]\" -titles \"[URL for Ancient Pages]\"", os.Args[0])
	fmt.Println("\n\n    -images \"[URL for files list]\"")
	fmt.Println("        Flag to download all of the images from a Mediawiki. See README for info on URL.")
	fmt.Println("\n    -titles \"[URL for Ancient Pages]\"")
	fmt.Println("        Flag to download all of the titles from a Mediawiki. See README for info on URL.")
	fmt.Println("\nEXAMPLES:")
	fmt.Printf("    %s -images \"http://wiki.foobar.com/wiki/index.php/Special:AncientPages&limit=500&offset=0\"\n", os.Args[0])
	fmt.Printf("    %s -titles \"http://wiki.foobar.com/wiki/index.php?limit=500&ilsearch=&title=Special%3AListFiles\"\n", os.Args[0])
	fmt.Println("\nView README for info for the proper URLs.")
	os.Exit(code)
}

func setupFlags(f *flag.FlagSet) {
	//Overiding os.Flags usage to use the show_usage Function
	f.Usage = func() {
		show_usage(0)
	}
}

func main() {
	//Parsethrough cmd line args to see which operation the user chose. If no operation
	//or incorrect flag, will show usage and exit. If -images is passed in prog will
	//find and download all images if given the correct link. If -titles is given
	//prog will find all titles to all pages in mediawiki if proper link is given.

	imagePtr := flag.String("images", "None", "a URL to point to the all files list on a mediawiki. See README for more info.")
	titlePtr := flag.String("titles", "None", "a URL to point to the oldest pages list on a mediawiki. See README for more info.")
	setupFlags(flag.CommandLine)
	flag.Parse()

	if len(os.Args) == 1 {
		show_usage(1)
	} else if len(os.Args) > 5 {
		show_usage(2)
	}

	if *imagePtr != "None" {
		strPtrValue := *imagePtr
		imageMain(strPtrValue)
	}
	if *titlePtr != "None" {
		titlePtrValue := *titlePtr
		titleMain(titlePtrValue)
	}

}
