package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/andybons/hipchat"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	cmdPrefix = "/giphy "
)

var (
	roomName  = flag.String("room", "", "The room that Buster should be active in.")
	authToken = flag.String("token", "", "The Hipchat auth token to use.")

	lastTime time.Time
)

type GiphyImageData struct {
	URL    string
	Width  string
	Height string
	Size   string
	Frames string
}

type GiphyGif struct {
	Type               string
	Id                 string
	URL                string
	Tags               string
	BitlyGifURL        string `json:"bitly_gif_url"`
	BitlyFullscreenURL string `json:"bitly_fullscreen_url"`
	BitlyTiledURL      string `json:"bitly_tiled_url"`
	Images             struct {
		Original               GiphyImageData
		FixedHeight            GiphyImageData `json:"fixed_height"`
		FixedHeightStill       GiphyImageData `json:"fixed_height_still"`
		FixedHeightDownsampled GiphyImageData `json:"fixed_height_downsampled"`
		FixedWidth             GiphyImageData `json:"fixed_width"`
		FixedwidthStill        GiphyImageData `json:"fixed_width_still"`
		FixedwidthDownsampled  GiphyImageData `json:"fixed_width_downsampled"`
	}
}

func main() {
	flag.Parse()

	if len(*authToken) == 0 || len(*roomName) == 0 {
		log.Fatal("usage: buster -token=<hipchat token> -room=<room name>")
	}

	lastTime = time.Now()
	c := hipchat.Client{AuthToken: *authToken}

	roomId := ""
	l, err := c.RoomList()
	if err != nil {
		log.Fatalf("RoomList: expected no error, but got %q", err)
	}
	for _, room := range l {
		if strings.ToLower(room.Name) == strings.ToLower(*roomName) {
			roomId = strconv.Itoa(room.Id)
			break
		}
	}
	if len(roomId) == 0 {
		log.Fatalf("No room was found with the name %q", *roomName)
	}
	for {
		time.Sleep(5 * time.Second)
		hist, err := c.RoomHistory(roomId, "recent", "EST")
		if err != nil {
			log.Printf("RoomHistory: Expected no error, but got %q", err)
		}
		for _, m := range hist {
			t, err := m.Time()
			if err != nil {
				log.Println(err)
				continue
			}
			if t.After(lastTime) {
				msg := m.Message
				if strings.HasPrefix(strings.ToLower(msg), cmdPrefix) {
					rockGiphy(msg[len(cmdPrefix):], &c)
				}
				log.Printf("Updating lastTime to %v", t)
				lastTime = t
			}
		}
	}
}

func rockGiphy(q string, c *hipchat.Client) {
	log.Printf("Searching for %q", q)
	url := fmt.Sprintf("http://api.giphy.com/v1/gifs/search?q=%s&api_key=dc6zaTOxFJmzC", url.QueryEscape(q))
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}
	giphyResp := &struct{ Data []GiphyGif }{}
	if err := json.Unmarshal(body, giphyResp); err != nil {
		log.Println(err)
		return
	}
	msg := "NO RESULTS. I’M A MONSTER."
	if len(giphyResp.Data) > 0 {
		msg = fmt.Sprintf("%s: %s", q, giphyResp.Data[rand.Intn(len(giphyResp.Data))].Images.Original.URL)
	}

	req := hipchat.MessageRequest{
		RoomId:        *roomName,
		From:          "BUSTER",
		Message:       msg,
		Color:         hipchat.ColorRandom,
		MessageFormat: hipchat.FormatText,
		Notify:        true,
	}
	if err := c.PostMessage(req); err != nil {
		log.Printf("Expected no error, but got %q", err)
	}
}
