package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/html"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

const userAgent = "https://sean.mcgivern.me.uk/the-hundred-combined-table/"

type Table struct {
	Rows Rows
}

type Rows []Row

func (r Rows) Len() int {
	return len(r)
}

func (r Rows) Less(i int, j int) bool {
	if r[i].Combined().Points() == r[j].Combined().Points() {
		return r[i].Combined().NetRunRate() > r[j].Combined().NetRunRate()
	} else {
		return r[i].Combined().Points() > r[j].Combined().Points()
	}
}

func (r Rows) Swap(i int, j int) {
	r[i], r[j] = r[j], r[i]
}

type Row struct {
	Team  string
	Women RowSection
	Men   RowSection
}

func (r Row) Combined() RowSection {
	return RowSection{
		Played:       r.Women.Played + r.Men.Played,
		Won:          r.Women.Won + r.Men.Won,
		Lost:         r.Women.Lost + r.Men.Lost,
		Tied:         r.Women.Tied + r.Men.Tied,
		NoResult:     r.Women.NoResult + r.Men.NoResult,
		BattingRuns:  r.Women.BattingRuns + r.Men.BattingRuns,
		BattingOvers: r.Women.BattingOvers + r.Men.BattingOvers,
		BowlingRuns:  r.Women.BowlingRuns + r.Men.BowlingRuns,
		BowlingOvers: r.Women.BowlingOvers + r.Men.BowlingOvers,
	}
}

type RowSection struct {
	Played       int
	Won          int
	Lost         int
	Tied         int
	NoResult     int
	BattingRuns  int
	BattingOvers float64
	BowlingRuns  int
	BowlingOvers float64
}

func (s RowSection) Points() int {
	return s.Won*2 + s.Tied + s.NoResult
}

func (s RowSection) NetRunRate() float64 {
	return (float64(s.BattingRuns) / s.BattingOvers) - (float64(s.BowlingRuns) / s.BowlingOvers)
}

// Yeah :-(
type CricinfoJson struct {
	Props struct {
		PageProps struct {
			Data struct {
				PageData struct {
					Content struct {
						Standings struct {
							Groups []struct {
								Name      string     `json:"name"`
								TeamStats []TeamStat `json:"teamStats"`
							} `json:"groups"`
						} `json:"standings"`
					} `json:"content"`
				} `json:"pageData"`
			} `json:"data"`
		} `json:"pageProps"`
	} `json:"props"`
}

type TeamStat struct {
	TeamInfo struct {
		Name string `json:"name"`
	} `json:"teamInfo"`
	MatchesPlayed   string `json:"matchesPlayed"`
	MatchesWon      int    `json:"matchesWon"`
	MatchesLost     int    `json:"matchesLost"`
	MatchesTied     int    `json:"matchesTied"`
	MatchesDrawn    int    `json:"matchesDrawn"`
	MatchesNoResult int    `json:"matchesNoResult"`
	For             string `json:"for"`
	Against         string `json:"against"`
}

func innerText(n *html.Node) (string, bool) {
	if n.Type == html.TextNode {
		return n.Data, true
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result, ok := innerText(c)
		if ok {
			return result, ok
		}
	}
	return "", false
}

func getTableJson(url string) string {
	client := http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", userAgent)

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	doc, err := html.Parse(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	ret := ""

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			for _, a := range n.Attr {
				if a.Key == "id" && a.Val == "__NEXT_DATA__" {
					ret, _ = innerText(n)
					break
				}
			}
		}
		if ret == "" {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}
	}
	f(doc)

	return ret
}

func getRowSections(url string) map[string]RowSection {
	sections := make(map[string]RowSection)
	results := CricinfoJson{}
	tableJson := getTableJson(url)
	err := json.Unmarshal([]byte(tableJson), &results)
	if err != nil {
		log.Fatal(err)
	}

	teamStats := results.Props.PageProps.Data.PageData.Content.Standings.Groups[0].TeamStats

	for i := range teamStats {
		stats := teamStats[i]
		played, err := strconv.Atoi(stats.MatchesPlayed)
		if err != nil {
			log.Fatal(err)
		}

		battingRuns, battingOvers := parseNrr(stats.For)
		bowlingRuns, bowlingOvers := parseNrr(stats.Against)

		sections[stats.TeamInfo.Name] = RowSection{
			Played:       played,
			Won:          stats.MatchesWon,
			Lost:         stats.MatchesLost,
			Tied:         stats.MatchesTied,
			NoResult:     stats.MatchesNoResult,
			BattingRuns:  battingRuns,
			BattingOvers: battingOvers,
			BowlingRuns:  bowlingRuns,
			BowlingOvers: bowlingOvers,
		}
	}

	return sections
}

func parseNrr(nrr string) (int, float64) {
	split := strings.Split(nrr, "/")

	runs, err := strconv.Atoi(split[0])
	if err != nil {
		log.Fatal(err)
	}

	// Five-ball overs, so this can be a decimal if we double the part after
	// the decimal point
	oversSplit := strings.Split(split[1], ".")
	balls, err := strconv.Atoi(oversSplit[1])
	if err != nil {
		log.Fatal(err)
	}

	overs, err := strconv.ParseFloat(fmt.Sprintf("%s.%d", oversSplit[0], balls*2), 64)
	if err != nil {
		log.Fatal(err)
	}

	return runs, overs
}

func main() {
	template := template.Must(template.New("index.html").ParseFiles("template/index.html"))

	women := getRowSections("https://www.espncricinfo.com/series/the-hundred-women-s-competition-2021-1252659/points-table-standings")
	men := getRowSections("https://www.espncricinfo.com/series/the-hundred-men-s-competition-2021-1252040/points-table-standings")

	rows := make(Rows, 0)
	for team, womenRow := range women {
		rows = append(rows, Row{
			Team:  team,
			Women: womenRow,
			Men:   men[team],
		})
	}

	sort.Sort(rows)

	template.Execute(os.Stdout, Table{
		Rows: rows,
	})
}
