package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

func getStatusJson(url string) (map[string]interface{}, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var status map[string]interface{}
	json.Unmarshal(body, &status)
	return status, nil
}

func main() {
	var (
		apiToken       = flag.String("apitoken", "", "Telegram API Token")
		spacestatusUrl = flag.String("spacestatusurl", "", "The URL to the space status file")
	)
	flag.Parse()

	if len(*apiToken) == 0 || len(*spacestatusUrl) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	b, err := tb.NewBot(tb.Settings{
		Token: *apiToken,
		// You can also set custom API URL. If field is empty it equals to "https://api.telegram.org"
		// URL:    "",
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
		return
	}

	b.Handle("/status", func(m *tb.Message) {
		status, err := getStatusJson(*spacestatusUrl)
		if err != nil {
			b.Send(m.Chat, "Oops... something went wrong. :(")
			return
		}
		b.Send(m.Chat, fmt.Sprintf("Hosts im Wifi: %.0f, Temp: %.1f°C, Tuer: %s, Lautstaerke: %.0f.", status["online"], status["temperature"], status["door"], status["sound"]))
	})

	b.Start()
}
