package main

import (
	"fmt"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/torbenconto/plutus/v2"
)

// List of US market index
var marketIndicators = []string{"^DJI", "^GSPC", "^IXIC"}

var marketOverviewStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	Padding(0, 1).
	Align(lipgloss.Center)

var containerStyle = lipgloss.NewStyle().Width(80).AlignHorizontal(lipgloss.Center)

var headerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("4")).
	Bold(true).
	AlignHorizontal(lipgloss.Center).
	Padding(1, 0, 1, 0)

var (
	negativeChangeColor = "#FF0000"
	positiveChangeColor = "#00ff00"
	neutralChangeColor  = "#454545"
)

var sectors = map[string][]string{
	"Finance": {
		"JPM", "GS", "BAC", "WFC", "C", "AXP", "BRK.B", "BLK", "V", "SCHW", "XLF",
	},
}

type SectorOverview struct {
	AverageChangePercent     float64
	Average52wkChangePercent float64
}

func getSectorData() (map[string]SectorOverview, error) {
	sectorOverview := make(map[string]SectorOverview)
	var wg sync.WaitGroup
	mu := sync.Mutex{} // To ensure safe concurrent updates to sectorOverview

	for sector, tickers := range sectors {
		wg.Add(1)
		go func(sector string, tickers []string) {
			defer wg.Done()

			var totalChangePercent, total52wkChangePercent float64
			var count int

			for _, ticker := range tickers {
				data, err := plutus.GetQuote(ticker)
				if err != nil {
					fmt.Printf("Error fetching data for %s: %v\n", ticker, err)
					continue
				}
				totalChangePercent += data.RegularMarketChangePercent
				total52wkChangePercent += data.FiftyTwoWeekChangePercent
				count++
			}

			if count > 0 {
				mu.Lock()
				sectorOverview[sector] = SectorOverview{
					AverageChangePercent:     totalChangePercent / float64(count),
					Average52wkChangePercent: total52wkChangePercent / float64(count),
				}
				mu.Unlock()
			}
		}(sector, tickers)
	}

	wg.Wait()

	return sectorOverview, nil
}

func main() {
	var boxes []string

	sectorDataChan := make(chan map[string]SectorOverview)
	errChan := make(chan error)

	go func() {
		sectorData, err := getSectorData()
		if err != nil {
			errChan <- err
			return
		}
		sectorDataChan <- sectorData
	}()

	for _, ticker := range marketIndicators {
		quote, err := plutus.GetQuote(ticker)
		if err != nil {
			panic(err)
		}

		var color lipgloss.Color

		switch {
		case quote.RegularMarketChangePercent > 0:
			color = lipgloss.Color(positiveChangeColor)
		case quote.RegularMarketChangePercent < 0:
			color = lipgloss.Color(negativeChangeColor)
		}

		overviewTicker := marketOverviewStyle.Render(fmt.Sprintf(
			"%s\n $%.2f %s",
			quote.Ticker,
			quote.RegularMarketPrice,
			lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("%.2f%%", quote.RegularMarketChangePercent)),
		))

		boxes = append(boxes, overviewTicker)
	}

	header := headerStyle.Render("Oikonomia")

	marketOverviewRow := lipgloss.JoinHorizontal(lipgloss.Top, boxes...)

	content := fmt.Sprintf("%s\n%s", header, marketOverviewRow)
	layout := containerStyle.Render(content)

	fmt.Println(layout)

	select {
	case sectorData := <-sectorDataChan:
		// Process the sectorData
		for sector, summary := range sectorData {
			fmt.Println(sector, summary.AverageChangePercent)
		}

	case err := <-errChan:
		panic(err)
	}
}
