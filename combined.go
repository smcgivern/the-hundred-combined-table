package main

import (
	"encoding/json"
	"fmt"
	"github.com/patrickmn/go-cache"
	"golang.org/x/net/html"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const userAgent = "https://sean.mcgivern.me.uk/the-hundred-combined-table/"
const defaultExpiration = 10 * time.Minute
const currentYear = "2022"
const womensTable = "https://www.espncricinfo.com/series/the-hundred-women-s-competition-2022-1299144/points-table-standings"
const mensTable = "https://www.espncricinfo.com/series/the-hundred-men-s-competition-2022-1299141/points-table-standings"

var c *cache.Cache

var previousYears = map[string]Rows{
	"2021": []Row{
		Row{"Brave",
			RowSection{8, 7, 1, 0, 0, 933, 138, 850, 149},
			RowSection{8, 5, 2, 0, 1, 983, 134.8, 990, 136.4},
		},
		Row{"Phoenix",
			RowSection{8, 4, 4, 0, 0, 1051, 154, 1029, 155},
			RowSection{8, 6, 2, 0, 0, 1202, 147.8, 1085, 154},
		},
		Row{"Invincibles",
			RowSection{8, 4, 3, 0, 1, 801, 136.4, 820, 140},
			RowSection{8, 4, 3, 0, 1, 975, 130.6, 956, 130.2},
		},
		Row{"Rockets",
			RowSection{8, 3, 4, 0, 1, 878, 138.4, 904, 136.2},
			RowSection{8, 5, 3, 0, 0, 1070, 145.2, 1084, 147.8},
		},
		Row{"N S-Chargers",
			RowSection{8, 3, 4, 0, 1, 862, 132, 849, 129.2},
			RowSection{8, 3, 4, 0, 1, 1054, 139.4, 935, 132.6},
		},
		Row{"Originals",
			RowSection{8, 3, 4, 0, 1, 823, 128.6, 835, 130.8},
			RowSection{8, 2, 4, 0, 2, 780, 113.6, 860, 119},
		},
		Row{"Spirit",
			RowSection{8, 4, 4, 0, 0, 934, 150, 963, 155.8},
			RowSection{8, 1, 6, 0, 1, 944, 140, 1019, 138},
		},
		Row{"Fire",
			RowSection{8, 2, 6, 0, 0, 927, 157.6, 959, 139},
			RowSection{8, 3, 5, 0, 0, 1148, 159.6, 1227, 153},
		},
	},
}

type Table struct {
	Rows          Rows
	GeneratedAt   string
	PreviousYears map[string]Rows
	Year          string
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
		AppPageProps struct {
			Data struct {
				Content struct {
					Standings struct {
						Groups []struct {
							Name      string     `json:"name"`
							TeamStats []TeamStat `json:"teamStats"`
						} `json:"groups"`
					} `json:"standings"`
				} `json:"content"`
			} `json:"data"`
		} `json:"appPageProps"`
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

	teamStats := results.Props.AppPageProps.Data.Content.Standings.Groups[0].TeamStats

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

func getRows(c *cache.Cache) (Rows, time.Time) {
	key := "Rows"
	fromCache, expires, found := c.GetWithExpiration(key)

	if found {
		return fromCache.(Rows), expires
	}

	women := getRowSections(womensTable)
	men := getRowSections(mensTable)

	rows := make(Rows, 0)
	for team, womenRow := range women {
		rows = append(rows, Row{
			Team:  team,
			Women: womenRow,
			Men:   men[team],
		})
	}

	sort.Sort(rows)

	c.Set(key, rows, defaultExpiration)

	return rows, time.Now().Add(defaultExpiration)
}

func logRequests(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func table(c *cache.Cache) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		year := r.FormValue("year")
		template := template.Must(template.New("index.html").ParseFiles("template/index.html"))

		var rows Rows
		var generatedAt string

		if year == "" {
			var expires time.Time

			rows, expires = getRows(c)
			generatedAt = expires.Add(defaultExpiration * -1).Format("2006-01-02 15:04:05 MST")
			year = currentYear
		} else {
			rows = previousYears[year]
			generatedAt = ""
		}

		template.Execute(w, Table{
			Rows:          rows,
			GeneratedAt:   generatedAt,
			PreviousYears: previousYears,
			Year:          year,
		})
	}
}

func main() {
	c = cache.New(defaultExpiration, 5*time.Minute)

	port, exists := os.LookupEnv("PORT")

	if !exists {
		port = "8080"
	}

	http.HandleFunc("/", table(c))

	log.Fatal(http.ListenAndServe(fmt.Sprintf("localhost:%s", port), logRequests(http.DefaultServeMux)))
}
