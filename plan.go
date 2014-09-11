package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	// "unicode/utf8"
	"sync"
	"unsafe"
	// "io"
	"container/list"
	"flag"
	"os"
)

type Auth_obj struct {
	Access_token string
}

type Tweet struct {
	Text string
}

type TweetCollection struct {
	Tweets []Tweet
}

func main() {
	introText := "SIMPLE TWITTER REFORMATTER \n (╯°□°）╯︵ ┻━┻) \n"
	fmt.Printf(introText)

	key := flag.String("key", "nokey", "Twitter consumer key")
	secret := flag.String("sec", "nosecret", "Twitter consumer secret")
	debug := flag.Bool("debug", false, "Debug logging level")
	numTweets := flag.Int("num", 3, "Number of tweets to retrieve")

	flag.Parse()

	access_token, err := getBearerToken(*key, *secret, *debug)
	if err != nil || access_token == "" {
		log.Fatal("Could not retrieve token to make twitter API request")
		os.Exit(1)
	}

	// Create a very basic channel with tweets getting passed into the expander
	// Wait for it to finish executing before quiting.
	var tweetChannel chan string = make(chan string)
	var wg sync.WaitGroup
	wg.Add(1)
	go tweetRetriever(access_token, *numTweets, tweetChannel, &wg, *debug)
	go textExpander(tweetChannel)
	wg.Wait()
}

func tweetRetriever(access_token string, numTweets int, tweetChannel chan string, wg *sync.WaitGroup, debug bool) {
	fmt.Println("Getting tweets")
	tweets, err := getTweets(numTweets, "iamdevloper", access_token, debug)
	if err != nil || unsafe.Sizeof(tweets) == 0 {
		log.Fatal("Could not make twitter API request")
		os.Exit(1)
	}
	for _, tweet := range tweets.Tweets {
		tweetChannel <- tweet.Text
	}
	wg.Done()
}

func textExpander(tweetChannel chan string) {
	for {
		msg := <-tweetChannel
		fmt.Println(expandText(msg))
	}
}

// @WARNING: font to explode to must all have same height.
func expandText(toExpand string) string {
	// First filter out newlines from original string
	toExpand = strings.Replace(toExpand, "\n", " ", -1)
	// create a list of large letters to format
	// Using a list instead of []string b/c utf8.RuneCountInString mismatches
	// the length with range when seeing fancy runes
	largeLetters := list.New()
	for _, r := range toExpand {
		char := string(r)
		largeLetters.PushBack(getLargeCharacter(char))
	}

	// Now combine characters in the same row
	// Get num rows from a character (janky, @TODO:search for max row height)
	dummyLetter := strings.Split(getLargeCharacter("a"), "\n")
	rows := make([]string, len(dummyLetter))
	for largeLetter := largeLetters.Front(); largeLetter != nil; largeLetter = largeLetter.Next() {
		splitStrings := strings.Split(largeLetter.Value.(string), "\n")
		for rowIndex, splitString := range splitStrings {
			rows[rowIndex] += "" + splitString
		}
	}

	// construct final string and add newlines at end of each row
	finalString := ""
	for _, row := range rows {
		finalString += row + " \n "
	}
	return finalString
}

// TESTS: make sure empty client creds fail
func getBearerToken(key string, secret string, debug bool) (string, error) {
	data := []byte(key + ":" + secret)
	encodedKey := base64.StdEncoding.EncodeToString(data)
	apiUrl := "https://api.twitter.com/oauth2/token"
	authReqValues := url.Values{}
	authReqValues.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", apiUrl, bytes.NewBufferString(authReqValues.Encode()))
	req.Header.Add("Authorization", "Basic "+encodedKey)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	testDump, err := httputil.DumpRequest(req, true)
	if err == nil && debug == true {
		fmt.Printf(" \n REQUEST: %s \n", testDump)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	defer resp.Body.Close()
	dumpResponse, err := httputil.DumpResponse(resp, true)
	if err == nil && debug == true {
		fmt.Printf("\n RESPONSE: %s \n", dumpResponse)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	var auth_data Auth_obj
	err = json.Unmarshal(body, &auth_data)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	fmt.Printf("access token! %s", auth_data.Access_token)
	return auth_data.Access_token, nil
}

// @TODO add in gzip for speed
// TESTS: zero numTweets fails
// TESTS: empty username fails
// TESTS: no token fails
func getTweets(numTweets int, username string, token string, debug bool) (TweetCollection, error) {
	// @TODO: replace this with url.Values
	numTweetsStr := strconv.Itoa(numTweets)
	apiUrl := "https://api.twitter.com/1.1/statuses/user_timeline.json?count=" + numTweetsStr + "&screen_name=" + username

	req, err := http.NewRequest("GET", apiUrl, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	// req.Header.Add("Accept-Encoding","gzip")
	testDump, err := httputil.DumpRequest(req, true)
	if err == nil && debug == true {
		fmt.Printf("\n REQUEST: %s \n", testDump)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
		return TweetCollection{}, err
	}
	defer resp.Body.Close()
	dumpResponse, err := httputil.DumpResponse(resp, true)
	if err == nil && debug == true {
		fmt.Printf("\n RESPONSE: %s \n", dumpResponse)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
		return TweetCollection{}, err
	}
	var tweetdump TweetCollection
	// @TODO: errors in unmarshal don't pass up, fix this.
	err = json.Unmarshal(body, &tweetdump.Tweets)
	if err != nil {
		log.Fatal(err)
		return TweetCollection{}, err
	}

	return tweetdump, nil
}

func TestHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
}

// Previously this made text MASSIVE, but line breaks on the command line
// broke it, so I'm backing up to something super simple like this. It will work
// with massive multi-line text blocks for each character.
// @TODO: implement figlet or something nicer to do this.
func getLargeCharacter(char string) string {
	return `
#
` + char + `
#`
}
