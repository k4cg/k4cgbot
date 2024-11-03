package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mb-14/gomarkov"

	tb "gopkg.in/tucnak/telebot.v2"
)

type Messages struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	ID       int64  `json:"id"`
	Messages []struct {
		ID      int      `json:"id"`
		Type    string   `json:"type"`
		Date    string   `json:"date"`
		Actor   string   `json:"actor"`
		ActorID int64    `json:"actor_id"`
		Action  string   `json:"action"`
		Title   string   `json:"title"`
		Members []string `json:"members"`
		Text    string   `json:"text"`
	} `json:"messages"`
}

func loadMarkovCorpus(chatHistoryFile string) *gomarkov.Chain {
	var messages Messages
	chain := gomarkov.NewChain(1)

	// init random seed
	rand.Seed(time.Now().Unix())

	// Parse json
	jsonFile, _ := os.Open(chatHistoryFile)
	byteValue, _ := io.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &messages)

	// Add Lines to Chain
	for i := 0; i < len(messages.Messages); i++ {
		line := messages.Messages[i].Text
		if len(line) > 1 {
			chain.Add(strings.Split(line, " "))
		}
	}

	return chain
}

func getMarkovSentence(chain *gomarkov.Chain) string {
	tokens := []string{gomarkov.StartToken}
	for tokens[len(tokens)-1] != gomarkov.EndToken {
		next, _ := chain.Generate(tokens[(len(tokens) - 1):])
		tokens = append(tokens, next)
	}

	return strings.Join(tokens[1:len(tokens)-1], " ")
}

func getStatusJson(url string) (map[string]interface{}, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var status map[string]interface{}
	json.Unmarshal(body, &status)
	return status, nil
}

func main() {

	// Arguments
	var (
		apiToken        = flag.String("apitoken", "", "Telegram API Token")
		spacestatusUrl  = flag.String("spacestatusurl", "", "The URL to the space status file")
		chatHistoryFile = flag.String("chathistoryfile", "", "The JSON Telegram Chat Export to build markov chains from")
	)
	flag.Parse()

	// Validate Arguments
	if len(*apiToken) == 0 || len(*spacestatusUrl) == 0 || len(*chatHistoryFile) == 0 {
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

	// Load and parse the telegram chat corpus at startup
	chain := loadMarkovCorpus(*chatHistoryFile)

	// K4CG Spacestatus in Channel
	b.Handle("/status", func(m *tb.Message) {
		status, err := getStatusJson(*spacestatusUrl)
		if err != nil {
			b.Send(m.Chat, "Oops... something went wrong. :(")
			return
		}

		state := status["state"].(map[string]interface{})

		var doorstate string
		if state["open"] == true {
			doorstate = "offen"
		} else {
			doorstate = "geschlossen"
		}

		b.Send(m.Chat, fmt.Sprintf("Tuer: %s", doorstate))
	})

	// Markov Chain output in Channel
	b.Handle("/sprachassistentin", func(m *tb.Message) {
		sentence := getMarkovSentence(chain)
		b.Send(m.Chat, sentence)
	})

	b.Start()
}
