package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/dustin/go-humanize"
	"github.com/torbenconto/plutus/v2"
	"golang.org/x/term"
)

var marketIndicators = []string{"^DJI", "^GSPC", "^IXIC"}

var (
	marketOverviewStyle lipgloss.Style
	containerStyle      lipgloss.Style
	entryStyle          = lipgloss.NewStyle().Align(lipgloss.Left)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("4")).
			Bold(true).
			AlignHorizontal(lipgloss.Center).
			Padding(1, 0, 1, 0)

	sectorsStyle = lipgloss.NewStyle().
			AlignHorizontal(lipgloss.Left).
			Padding(1, 0, 1, 0)

	subHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("4")).
			Bold(true).
			AlignHorizontal(lipgloss.Center).
			Italic(true).
			Padding(0, 0, 1, 0)

	stockTopRowStyle = lipgloss.NewStyle().
				AlignHorizontal(lipgloss.Center).
				Bold(true).
				Padding(0, 1)
)

var (
	negativeChangeColor = "#FF0000"
	positiveChangeColor = "#00FF00"
	neutralChangeColor  = "#454545"
)

var sectors = map[string][]string{
	"Finance":                {"JPM", "GS", "BAC", "WFC", "C", "AXP", "BRK.B", "BLK", "V", "SCHW", "XLF"},
	"Technology":             {"AAPL", "MSFT", "GOOGL", "NVDA", "META", "AMD", "INTC", "TSM", "CRM", "ORCL", "XLK"},
	"Healthcare":             {"JNJ", "PFE", "MRK", "UNH", "ABBV", "TMO", "ABT", "LLY", "BMY", "CVS", "XLV"},
	"Energy":                 {"XOM", "CVX", "COP", "SLB", "PSX", "EOG", "VLO", "MPC", "KMI", "HAL", "XLE"},
	"Consumer Discretionary": {"AMZN", "TSLA", "HD", "NKE", "SBUX", "MCD", "LOW", "TGT", "BKNG", "ROST", "XLY"},
	"Consumer Staples":       {"PG", "KO", "PEP", "WMT", "COST", "MO", "PM", "CL", "KHC", "KR", "XLP"},
	"Industrials":            {"BA", "CAT", "GE", "UPS", "UNP", "DE", "MMM", "LMT", "RTX", "NOC", "XLI"},
	"Utilities":              {"NEE", "DUK", "SO", "D", "AEP", "EXC", "SRE", "PEG", "XEL", "ED", "XLU"},
	"Materials":              {"LIN", "SHW", "NEM", "DD", "FCX", "APD", "ECL", "NUE", "MLM", "ALB", "XLB"},
	"Real Estate":            {"PLD", "AMT", "CCI", "EQIX", "O", "SPG", "DLR", "WELL", "AVB", "VTR", "XLRE"},
	"Communication Services": {"GOOGL", "META", "DIS", "NFLX", "TMUS", "VZ", "T", "CHTR", "EA", "XLC"},
}

type SectorOverview struct {
	AverageChangePercent     float64
	Average52wkChangePercent float64
}

func getSectorData() (map[string]SectorOverview, error) {
	sectorOverview := make(map[string]SectorOverview)
	var wg sync.WaitGroup
	mu := sync.Mutex{}

	for sector, tickers := range sectors {
		wg.Add(1)
		go func(sector string, tickers []string) {
			defer wg.Done()

			var totalChange, total52wkChange float64
			var count int

			for _, ticker := range tickers {
				data, err := plutus.GetQuote(ticker)
				if err != nil {
					fmt.Printf("Error fetching %s: %v\n", ticker, err)
					continue
				}
				totalChange += data.RegularMarketChangePercent
				total52wkChange += data.FiftyTwoWeekChangePercent
				count++
			}

			if count > 0 {
				mu.Lock()
				sectorOverview[sector] = SectorOverview{
					AverageChangePercent:     totalChange / float64(count),
					Average52wkChangePercent: total52wkChange / float64(count),
				}
				mu.Unlock()
			}
		}(sector, tickers)
	}

	wg.Wait()
	return sectorOverview, nil
}

func marketOverview() {
	var boxes []string
	sectorDataChan := make(chan map[string]SectorOverview)
	errChan := make(chan error)

	go func() {
		data, err := getSectorData()
		if err != nil {
			errChan <- err
			return
		}
		sectorDataChan <- data
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
		default:
			color = lipgloss.Color(neutralChangeColor)
		}

		box := marketOverviewStyle.Render(fmt.Sprintf(
			"%s\n$%s %s",
			quote.Ticker,
			humanize.FormatFloat("#,###.##", quote.RegularMarketPrice),
			lipgloss.NewStyle().
				Foreground(color).
				Render(fmt.Sprintf("%.2f%%", quote.RegularMarketChangePercent)),
		))
		boxes = append(boxes, box)
	}

	marketOverviewRow := lipgloss.JoinHorizontal(lipgloss.Top, boxes...)

	var sectorRows []string
	select {
	case sectorData := <-sectorDataChan:
		for sector, summary := range sectorData {
			var sectorColor lipgloss.Color
			switch {
			case summary.AverageChangePercent > 0:
				sectorColor = lipgloss.Color(positiveChangeColor)
			case summary.AverageChangePercent < 0:
				sectorColor = lipgloss.Color(negativeChangeColor)
			default:
				sectorColor = lipgloss.Color(neutralChangeColor)
			}

			sectorRows = append(sectorRows, fmt.Sprintf("%-24s %s", sector+":", lipgloss.NewStyle().Foreground(sectorColor).Render(fmt.Sprintf("%.2f%%", summary.AverageChangePercent))))
		}
	case err := <-errChan:
		panic(err)
	}

	sectorComponent := sectorsStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, sectorRows...),
	)

	content := fmt.Sprintf("%s\n\n%s", marketOverviewRow, sectorComponent)
	layout := containerStyle.Render(content)

	fmt.Println(layout)
}

func stockData(ticker string) {
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
	default:
		color = lipgloss.Color(neutralChangeColor)
	}

	topRow := stockTopRowStyle.Render(fmt.Sprintf("%s   $%s   %s", quote.Ticker, humanize.FormatFloat("#,###.##", quote.RegularMarketPrice), lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("%.2f%%", quote.RegularMarketChangePercent))))

	var marketCap string
	if quote.MarketCap > 0 {
		marketCap = "$" + humanize.Comma(int64(quote.MarketCap))
	} else {
		marketCap = "n/a"
	}

	entries := []struct {
		label string
		value interface{}
	}{
		{"Open Price", fmt.Sprintf("$%.2f", quote.RegularMarketOpen)},
		{"High Price", fmt.Sprintf("$%.2f", quote.RegularMarketDayHigh)},
		{"Low Price", fmt.Sprintf("$%.2f", quote.RegularMarketDayLow)},
		{"52wk High", fmt.Sprintf("$%.2f", quote.FiftyTwoWeekHigh)},
		{"52wk Low", fmt.Sprintf("$%.2f", quote.FiftyTwoWeekLow)},
		{
			"52wk Change",
			lipgloss.NewStyle().
				Foreground(func() lipgloss.Color {
					switch {
					case quote.FiftyTwoWeekChangePercent > 0:
						return lipgloss.Color(positiveChangeColor)
					case quote.FiftyTwoWeekChangePercent < 0:
						return lipgloss.Color(negativeChangeColor)
					default:
						return lipgloss.Color(neutralChangeColor)
					}
				}()).
				Render(fmt.Sprintf("%.2f%%", quote.FiftyTwoWeekChangePercent)),
		},
		{"Market Cap", marketCap},
		{"Volume", humanize.Comma(int64(quote.RegularMarketVolume))},
		{"Avg Volume", humanize.Comma(int64(quote.AverageDailyVolume3Month))},
	}

	numCols := 3
	numRows := (len(entries) + numCols - 1) / numCols

	rows := make([][]string, numRows)
	for r := 0; r < numRows; r++ {
		rows[r] = make([]string, numCols)
		for c := 0; c < numCols; c++ {
			idx := c*numRows + r
			if idx < len(entries) {
				rows[r][c] = fmt.Sprintf("%-6s: %v", entries[idx].label, entries[idx].value)
			} else {
				rows[r][c] = ""
			}
		}
	}

	tbl := table.New().
		Border(lipgloss.RoundedBorder()).
		StyleFunc(func(row, col int) lipgloss.Style {
			return entryStyle.Padding(1, 2)
		})

	for _, row := range rows {
		tbl.Row(row...)
	}

	fmt.Println(containerStyle.Render(topRow))
	fmt.Println(containerStyle.Render(tbl.Render()))
}

func main() {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		panic(err)
	}

	containerStyle = lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(width)

	marketOverviewStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Align(lipgloss.Center).
		Border(lipgloss.RoundedBorder())

	entryStyle = lipgloss.NewStyle().Align(lipgloss.Left)

	header := headerStyle.Render("Oikonomia")
	subheader := subHeaderStyle.Render("A Financial Market Analysis Tool")

	fmt.Println(containerStyle.Render(header))
	fmt.Println(containerStyle.Render(subheader))

	if len(os.Args) > 1 {
		stockData(os.Args[1])
		return
	}

	marketOverview()
}
