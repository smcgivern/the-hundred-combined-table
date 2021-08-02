package main

import (
	"html/template"
	"os"
)

type RowSection struct {
	Played       int
	Won          int
	Lost         int
	Tied         int
	NoResult     int
	Points       int
	BattingRuns  int
	BattingOvers int
	BowlingRuns  int
	BowlingOvers int
}

type Row struct {
	Team     string
	Combined RowSection
	Women    RowSection
	Men      RowSection
}

type Table struct {
	Rows []Row
}

func main() {
	template := template.Must(template.New("index.html").ParseFiles("template/index.html"))
	rows := []Row{
		Row{Team: "Welsh Fire"},
		Row{Team: "Oval Invincibles"},
		Row{Team: "London Spirit"},
	}

	template.Execute(os.Stdout, Table{
		Rows: rows,
	})
}
