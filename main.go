package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mb-14/gomarkov"

	tb "gopkg.in/telebot.v3"
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

type SpaceApi struct {
	State struct {
		Open       *bool  `json:"open"`
		LastChange *int64 `json:"lastchange"`
	} `json:"state"`
	Sensors struct {
		Temperature []struct {
			Location string  `json:"location"`
			Unit     string  `json:"unit"`
			Value    float64 `json:"value"`
		} `json:"temperature"`
		Humidity []struct {
			Location string  `json:"location"`
			Unit     string  `json:"unit"`
			Value    float64 `json:"value"`
		} `json:"humidity"`
		Carbondioxide []struct {
			Location string  `json:"location"`
			Unit     string  `json:"unit"`
			Value    float64 `json:"value"`
		} `json:"carbondioxide"`
	}
}

func loadMarkovCorpus(chatHistoryFile string) (chain *gomarkov.Chain, err error) {
	var messages Messages
	chain = gomarkov.NewChain(1)

	// Parse json
	jsonFile, err := os.Open(chatHistoryFile)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(byteValue, &messages)

	// Add Lines to Chain
	addedLine := false
	for i := 0; i < len(messages.Messages); i++ {
		line := messages.Messages[i].Text
		if len(line) > 1 {
			chain.Add(strings.Split(line, " "))
			addedLine = true
		}
	}

	if !addedLine {
		return nil, io.EOF
	}

	return chain, nil
}

func getMarkovSentence(chain *gomarkov.Chain) string {
	tokens := []string{gomarkov.StartToken}
	for tokens[len(tokens)-1] != gomarkov.EndToken {
		next, _ := chain.Generate(tokens[(len(tokens) - 1):])
		tokens = append(tokens, next)
	}

	return strings.Join(tokens[1:len(tokens)-1], " ")
}

func getStatusJson(url string) (status SpaceApi, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return status, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &status)
	return status, nil
}

func openSince(lastchange int64) string {
	diff := time.Now().Sub(time.Unix(lastchange, 0)).Round(time.Second)

	d := diff / (time.Hour * 24)
	diff -= d * (time.Hour * 24)
	h := diff / time.Hour
	diff -= h * time.Hour
	m := diff / time.Minute
	diff -= m * time.Minute
	s := diff / time.Second

	if d > 0 {
		return fmt.Sprintf("%dd%dh", d, h)
	} else if h > 0 {
		return fmt.Sprintf("%dh%dm", h, m)
	}
	return fmt.Sprintf("%dm%ds", m, s)
}

func statusToString(status SpaceApi, sensorLocation string) string {
	var info []string

	// Door status
	door := "Tür: "
	if status.State.Open != nil && status.State.LastChange != nil {
		if *status.State.Open == true {
			door += "offen"
		} else {
			door += "geschlossen"
		}
		door += fmt.Sprintf(" (seit %s)", openSince(*status.State.LastChange))
	} else {
		door += "unbekannt"
	}
	info = append(info, door)

	// Temperature sensor
	if len(status.Sensors.Temperature) > 0 {
		for _, temp := range status.Sensors.Temperature {
			if temp.Location == sensorLocation {
				info = append(info, fmt.Sprintf("Temperatur: %.1f%s", temp.Value, temp.Unit))
				break
			}
		}
	}

	// Humidity sensor
	if len(status.Sensors.Humidity) > 0 {
		for _, humid := range status.Sensors.Humidity {
			if humid.Location == sensorLocation {
				info = append(info, fmt.Sprintf("Luftfeuchtigkeit: %.0f%s", humid.Value, humid.Unit))
				break
			}
		}
	}

	// CO2 sensor
	if len(status.Sensors.Carbondioxide) > 0 {
		for _, co2 := range status.Sensors.Carbondioxide {
			if co2.Location == sensorLocation {
				info = append(info, fmt.Sprintf("CO2: %.0f%s", co2.Value, co2.Unit))
				break
			}
		}
	}

	return strings.Join(info[:], ", ")
}

const (
	ENV_APITOKEN  = "K4B_APITOKEN"
	ENV_SPACEAPI  = "K4B_SPACEAPI"
	ENV_CHATHIST  = "K4B_CHATHIST"
	ENV_SENSORLOC = "K4B_SENSORLOC"
)

func main() {
	// Commandline arguments
	var (
		apiToken        = flag.String("apitoken", "", fmt.Sprintf("Telegram API token (env %s)", ENV_APITOKEN))
		spacestatusUrl  = flag.String("spacestatusurl", "", fmt.Sprintf("The URL to the space status (env %s)", ENV_SPACEAPI))
		chatHistoryFile = flag.String("chathistoryfile", "", fmt.Sprintf("The JSON Telegram chat export to build markov chains from (env %s)", ENV_CHATHIST))
		sensorLocation  = flag.String("sensorlocation", "", fmt.Sprintf("Location of sensor information to add to the status message (env %s)", ENV_SENSORLOC))
	)
	flag.Parse()

	// Alternatively check for environment variables
	if len(*apiToken) == 0 {
		*apiToken = os.Getenv(ENV_APITOKEN)
	}
	if len(*spacestatusUrl) == 0 {
		*spacestatusUrl = os.Getenv(ENV_SPACEAPI)
	}
	if len(*chatHistoryFile) == 0 {
		*chatHistoryFile = os.Getenv(ENV_CHATHIST)
	}
	if len(*sensorLocation) == 0 {
		*sensorLocation = os.Getenv(ENV_SENSORLOC)
	}

	// Basic input check
	if len(*apiToken) == 0 || len(*spacestatusUrl) == 0 || len(*chatHistoryFile) == 0 || len(*sensorLocation) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	// Create new bot
	b, err := tb.NewBot(tb.Settings{
		Token:  *apiToken,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
		return
	}

	// Load and parse the telegram chat corpus at startup
	chain, err := loadMarkovCorpus(*chatHistoryFile)
	if err != nil {
		log.Fatalf("Failed to parse chat history file %s", *chatHistoryFile)
		return
	}

	// K4CG spacestatus in channel
	b.Handle("/status", func(c tb.Context) error {
		status, err := getStatusJson(*spacestatusUrl)
		if err != nil {
			return c.Send("Oops... something went wrong. :(")
		} else {
			return c.Send(statusToString(status, *sensorLocation))
		}
	})

	// Markov chain output in channel
	b.Handle("/sprachassistentin", func(c tb.Context) error {
		sentence := getMarkovSentence(chain)
		return c.Send(sentence)
	})

	b.Start()
}
