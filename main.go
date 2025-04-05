package main

import (
	"fmt"

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

func main() {
	var boxes []string

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
}
