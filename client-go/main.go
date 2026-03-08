// Pi Terminal World Monitor – Go client using tview.
// Fetches panel data from the backend; displays a responsive grid of panels.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	defaultBackendURL = "http://localhost:8000"
	defaultCycleSecs  = 8
	defaultGridCols   = 2
	defaultGridRows   = 2
	requestTimeout    = 25 * time.Second // weather fetches many cities; backend uses parallel fetches
)

var (
	httpClient = &http.Client{Timeout: requestTimeout}
)

type panelsList struct {
	Panels []string `json:"panels"`
	Status string   `json:"status"`
}

type worldClockResp struct {
	Status string     `json:"status"`
	UTC    string     `json:"utc"`
	Zones  []zoneInfo `json:"zones"`
}

type zoneInfo struct {
	Name string `json:"name"`
	Time string `json:"time"`
	Date string `json:"date"`
}

type weatherLoc struct {
	Name       string `json:"name"`
	Temp       string `json:"temp"`
	TempHigh   string `json:"temp_high"`
	TempLow    string `json:"temp_low"`
	Conditions string `json:"conditions"`
	WeatherCode int   `json:"weather_code"`
}

type weatherContinent struct {
	Name      string       `json:"name"`
	Locations []weatherLoc `json:"locations"`
}

type weatherResp struct {
	Status     string              `json:"status"`
	Message    string              `json:"message"`
	Continents []weatherContinent  `json:"continents"`
	Locations  []weatherLoc        `json:"locations"` // legacy flat list
}

type gsmRegion struct {
	Name     string   `json:"name"`
	Severity string   `json:"severity"`
	Events   []string `json:"events"`
}

type globalSituationMapResp struct {
	Status  string      `json:"status"`
	Source  string      `json:"source"`
	Regions []gsmRegion `json:"regions"`
	Message string      `json:"message"`
}

// Crypto panel (4 sub-panels: top cryptos, stablecoins, crypto news, btc-etf)
type cryptoCoin struct {
	Rank        int      `json:"rank"`
	Symbol      string   `json:"symbol"`
	Name        string   `json:"name"`
	Price       float64  `json:"price"`
	Price24hPct *float64 `json:"price_24h_pct"`
	Price7dPct  *float64 `json:"price_7d_pct"`
}

type cryptoTopResp struct {
	Status string       `json:"status"`
	Range  string       `json:"range"`
	Coins  []cryptoCoin `json:"coins"`
}

type cryptoStablecoin struct {
	Symbol       string  `json:"symbol"`
	Name         string  `json:"name"`
	Price        float64 `json:"price"`
	PegStatus    string  `json:"peg_status"`
	DeviationPct float64 `json:"deviation_pct"`
}

type cryptoStablecoinsResp struct {
	Status      string              `json:"status"`
	StatusLabel string              `json:"status_label"`
	MarketCapB  *float64            `json:"market_cap_b"`
	VolumeB     *float64            `json:"volume_b"`
	Coins       []cryptoStablecoin  `json:"coins"`
}

type cryptoBtcEtfEntry struct {
	Name     string  `json:"name"`
	Flows24h float64 `json:"flows_24h"`
	Aum      float64 `json:"aum"`
}

type cryptoBtcEtfResp struct {
	Status        string              `json:"status"`
	Message       string              `json:"message"`
	Etfs          []cryptoBtcEtfEntry `json:"etfs"`
	TotalFlows24h interface{}         `json:"total_flows_24h"`
	TotalAum      interface{}         `json:"total_aum"`
}

type cryptoNewsItem struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	PubDate string `json:"pub_date"`
}

type cryptoNewsResp struct {
	Status string          `json:"status"`
	Source string          `json:"source"`
	Items  []cryptoNewsItem `json:"items"`
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func fetchJSON(baseURL, path string, v interface{}) error {
	url := strings.TrimSuffix(baseURL, "/") + path
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func fetchPanels(baseURL string) ([]string, error) {
	var list panelsList
	if err := fetchJSON(baseURL, "/panels", &list); err != nil {
		return nil, err
	}
	if len(list.Panels) == 0 {
		return []string{"crypto", "weather", "global-situation-map", "world-clock"}, nil
	}
	return list.Panels, nil
}

func renderWorldClock(baseURL string) string {
	var d worldClockResp
	if err := fetchJSON(baseURL, "/panels/world-clock", &d); err != nil {
		return "No data"
	}
	if d.Status != "ok" {
		return "No data"
	}
	var b strings.Builder
	b.WriteString(d.UTC)
	b.WriteString("\n")
	for _, z := range d.Zones {
		b.WriteString(fmt.Sprintf("  %s: %s %s\n", z.Name, z.Date, z.Time))
	}
	return strings.TrimSuffix(b.String(), "\n")
}

var weatherStartTime time.Time

// weatherIcon returns a single Unicode symbol for WMO code (terminal-friendly).
func weatherIcon(code int) string {
	switch {
	case code == 0:
		return "☀"
	case code >= 1 && code <= 3:
		return "☁"
	case code == 45 || code == 48:
		return "🌫"
	case code >= 51 && code <= 67:
		return "🌧"
	case code >= 71 && code <= 86:
		return "❄"
	case code >= 95 && code <= 99:
		return "⛈"
	case code >= 80 && code <= 82:
		return "🌦"
	default:
		return "·"
	}
}

// tempColor returns tview color tag for heat map: cold→blue/cyan, mild→green, warm→yellow, hot→red.
func tempColor(tempStr string) string {
	if tempStr == "" || tempStr == "—" {
		return "[gray]"
	}
	n, err := strconv.Atoi(tempStr)
	if err != nil {
		return "[white]"
	}
	switch {
	case n < 10:
		return "[blue]"
	case n < 18:
		return "[cyan]"
	case n < 25:
		return "[green]"
	case n < 30:
		return "[yellow]"
	default:
		return "[red]"
	}
}

func renderWeather(baseURL string) string {
	var d weatherResp
	if err := fetchJSON(baseURL, "/panels/weather", &d); err != nil {
		return "No data"
	}
	if d.Status == "placeholder" && d.Message != "" {
		return d.Message
	}
	var locations []weatherLoc
	var continentName string
	var regionNum, totalRegions int
	if len(d.Continents) > 0 {
		// Time-based rotation: next continent every 4s (weather panel has its own 4s ticker)
		const weatherRegionCycleSecs = 4
		elapsed := int(time.Since(weatherStartTime).Seconds())
		idx := (elapsed / weatherRegionCycleSecs) % len(d.Continents)
		c := d.Continents[idx]
		continentName = c.Name
		locations = c.Locations
		regionNum = idx + 1
		totalRegions = len(d.Continents)
	} else if len(d.Locations) > 0 {
		locations = d.Locations
	}
	if len(locations) == 0 {
		return "No locations yet."
	}
	var b strings.Builder
	if continentName != "" {
		// Show region index so user can verify cycling (e.g. "2/8")
		b.WriteString(fmt.Sprintf(" [yellow]%s[-] [gray](%d/%d)[-]\n", continentName, regionNum, totalRegions))
	}
	for _, loc := range locations {
		icon := weatherIcon(loc.WeatherCode)
		tag := tempColor(loc.Temp)
		hi, lo := loc.TempHigh, loc.TempLow
		if hi == "" {
			hi = "—"
		}
		if lo == "" {
			lo = "—"
		}
		b.WriteString(fmt.Sprintf("  %s %s %s%s°[-] (%s°/ %s°) %s\n", icon, loc.Name, tag, loc.Temp, lo, hi, loc.Conditions))
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// fmtPrice formats price with commas and decimals for readability (e.g. 66825 → $66,825.00).
func fmtPrice(p float64) string {
	var s string
	if p >= 1 {
		s = fmt.Sprintf("%.2f", p)
		// Insert commas for thousands
		dot := strings.Index(s, ".")
		intPart := s
		if dot >= 0 {
			intPart = s[:dot]
		}
		var b strings.Builder
		for i, r := range intPart {
			if i > 0 && (len(intPart)-i)%3 == 0 {
				b.WriteString(",")
			}
			b.WriteRune(r)
		}
		if dot >= 0 {
			b.WriteString(s[dot:])
		}
		return "$" + b.String()
	}
	if p >= 0.01 {
		return fmt.Sprintf("$%.2f", p)
	}
	return fmt.Sprintf("$%.4f", p)
}

func pctColor(pct *float64) string {
	if pct == nil {
		return "[white]"
	}
	if *pct >= 0 {
		return "[green]"
	}
	return "[red]"
}

func renderCryptoTop(baseURL string, rangeStart int) string {
	var d cryptoTopResp
	path := fmt.Sprintf("/panels/crypto/top?range_start=%d", rangeStart)
	if err := fetchJSON(baseURL, path, &d); err != nil {
		return "No data"
	}
	if d.Status != "ok" || len(d.Coins) == 0 {
		return "No data"
	}
	var b strings.Builder
	for _, c := range d.Coins {
		p24 := "—"
		if c.Price24hPct != nil {
			p24 = fmt.Sprintf("%+.2f%%", *c.Price24hPct)
		}
		p7d := "—"
		if c.Price7dPct != nil {
			p7d = fmt.Sprintf("%+.2f%%", *c.Price7dPct)
		}
		tag24 := pctColor(c.Price24hPct)
		tag7d := pctColor(c.Price7dPct)
		b.WriteString(fmt.Sprintf("  %2d %s %s 24h %s%s[-] 7d %s%s[-]\n", c.Rank, c.Symbol, fmtPrice(c.Price), tag24, p24, tag7d, p7d))
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func renderCryptoStablecoins(baseURL string) string {
	var d cryptoStablecoinsResp
	if err := fetchJSON(baseURL, "/panels/crypto/stablecoins", &d); err != nil {
		return "No data"
	}
	if d.Status != "ok" {
		return "No data"
	}
	var b strings.Builder
	statusTag := "[green]"
	if d.StatusLabel != "Healthy" {
		statusTag = "[yellow]"
	}
	b.WriteString(fmt.Sprintf(" %s%s[-]\n", statusTag, d.StatusLabel))
	if d.MarketCapB != nil && d.VolumeB != nil {
		b.WriteString(fmt.Sprintf(" MCap: $%.1fB  Vol: $%.1fB\n", *d.MarketCapB, *d.VolumeB))
	}
	b.WriteString(" [gray]PEG HEALTH[-]\n")
	for _, c := range d.Coins {
		pegTag := "[green]"
		if c.PegStatus != "ON PEG" {
			pegTag = "[red]"
		}
		b.WriteString(fmt.Sprintf("  %s $%.4f %s%s[-] %+.2f%%\n", c.Symbol, c.Price, pegTag, c.PegStatus, c.DeviationPct))
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func renderCryptoBtcEtf(baseURL string) string {
	var d cryptoBtcEtfResp
	if err := fetchJSON(baseURL, "/panels/crypto/btc-etf", &d); err != nil {
		return "No data"
	}
	if len(d.Etfs) > 0 || d.TotalFlows24h != nil || d.TotalAum != nil {
		var b strings.Builder
		if d.TotalFlows24h != nil {
			if f, ok := d.TotalFlows24h.(float64); ok {
				tag := "[green]"
				if f < 0 {
					tag = "[red]"
				}
				b.WriteString(fmt.Sprintf("  Flows 24h: %s%+.0f[-] M\n", tag, f))
			}
		}
		if d.TotalAum != nil {
			if a, ok := d.TotalAum.(float64); ok {
				b.WriteString(fmt.Sprintf("  Total AUM: %.1f B\n", a))
			}
		}
		for _, e := range d.Etfs {
			b.WriteString(fmt.Sprintf("  %s flows %+.0f AUM %.1f B\n", e.Name, e.Flows24h, e.Aum))
		}
		return strings.TrimSuffix(b.String(), "\n")
	}
	if d.Message != "" {
		return d.Message
	}
	return "BTC ETF data TBD"
}

func renderCryptoNews(baseURL string) string {
	var d cryptoNewsResp
	if err := fetchJSON(baseURL, "/panels/crypto/news", &d); err != nil {
		return "No data"
	}
	if d.Status != "ok" || len(d.Items) == 0 {
		return "No headlines"
	}
	var b strings.Builder
	for i, it := range d.Items {
		if i >= 8 {
			break
		}
		title := it.Title
		if len(title) > 55 {
			title = title[:52] + "..."
		}
		b.WriteString(fmt.Sprintf("  %s\n", title))
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// renderCrypto is used only when crypto is a single panel (legacy); prefer crypto 3-subpanel layout.
func renderCrypto(baseURL string) string {
	return renderCryptoTop(baseURL, 1)
}

func renderGlobalSituationMap(baseURL string) string {
	var d globalSituationMapResp
	if err := fetchJSON(baseURL, "/panels/global-situation-map", &d); err != nil {
		return "No data"
	}
	if d.Status != "ok" && d.Status != "" {
		if d.Message != "" {
			return d.Message
		}
		return "No data"
	}
	var b strings.Builder
	for _, r := range d.Regions {
		// Severity color: critical=red, elevated=yellow, monitoring=cyan, normal=green (tview tags)
		sevTag := "[white]"
		switch strings.ToLower(r.Severity) {
		case "critical":
			sevTag = "[red]"
		case "elevated":
			sevTag = "[yellow]"
		case "monitoring":
			sevTag = "[cyan]"
		case "normal":
			sevTag = "[green]"
		}
		b.WriteString(fmt.Sprintf("  %s%s[-] %s", sevTag, r.Severity, r.Name))
		if len(r.Events) > 0 {
			b.WriteString(" · ")
			b.WriteString(strings.Join(r.Events, ", "))
		}
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func panelTitle(key string) string {
	switch key {
	case "crypto":
		return "Crypto"
	case "world-clock":
		return "World Clock"
	case "weather":
		return "Weather Watch"
	case "global-situation-map":
		return "Global Situation Map"
	default:
		return key
	}
}

func panelContent(baseURL, key string) string {
	switch key {
	case "crypto":
		return renderCrypto(baseURL)
	case "world-clock":
		return renderWorldClock(baseURL)
	case "weather":
		return renderWeather(baseURL)
	case "global-situation-map":
		return renderGlobalSituationMap(baseURL)
	default:
		return "?"
	}
}

func main() {
	weatherStartTime = time.Now()

	baseURL := getEnv("BACKEND_URL", defaultBackendURL)
	cycleSecs := defaultCycleSecs
	if s := os.Getenv("CYCLE_SECONDS"); s != "" {
		if n, err := fmt.Sscanf(s, "%d", &cycleSecs); n == 1 && err == nil && cycleSecs >= 1 && cycleSecs <= 120 {
			// use it
		}
	}
	panels, err := fetchPanels(baseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot reach backend %s: %v\n", baseURL, err)
		os.Exit(1)
	}

	cols := defaultGridCols
	rows := defaultGridRows
	if s := os.Getenv("GRID_COLS"); s != "" {
		if c, err := strconv.Atoi(s); err == nil && c >= 1 && c <= 6 {
			cols = c
		}
	}
	if s := os.Getenv("GRID_ROWS"); s != "" {
		if r, err := strconv.Atoi(s); err == nil && r >= 1 && r <= 6 {
			rows = r
		}
	}

	app := tview.NewApplication()

	n := cols * rows
	if n > len(panels) {
		for len(panels) < n {
			panels = append(panels, panels...)
		}
		panels = panels[:n]
	} else {
		panels = panels[:n]
	}

	// Grid columns: equal width per column; rows: equal height + 1 for footer
	gridCols := make([]int, cols)
	for i := 0; i < cols; i++ {
		gridCols[i] = -1
	}
	gridRows := make([]int, rows+1)
	for i := 0; i < rows; i++ {
		gridRows[i] = -1
	}
	gridRows[rows] = 1
	grid := tview.NewGrid().
		SetColumns(gridCols...).
		SetRows(gridRows...).
		SetBorders(false)

	// One TextView per cell (or a Flex for crypto with 4 sub-panels in 2×2)
	textViews := make([]*tview.TextView, n)
	var cryptoSubpanelViews [4]*tview.TextView
	for i := 0; i < n; i++ {
		key := panels[i]
		row, col := i/cols, i%cols
		if key == "crypto" {
			// Crypto: 4 bordered sub-panels in 2×2 (Top Cryptos | Stable Coins; Crypto News | BTC ETF)
			tvTop := tview.NewTextView().SetDynamicColors(true).SetText(renderCryptoTop(baseURL, 1))
			tvTop.SetBorder(true).SetTitle(" Top cryptos (1-12) ")
			tvStable := tview.NewTextView().SetDynamicColors(true).SetText(renderCryptoStablecoins(baseURL))
			tvStable.SetBorder(true).SetTitle(" Stablecoins ")
			tvNews := tview.NewTextView().SetDynamicColors(true).SetText(renderCryptoNews(baseURL))
			tvNews.SetBorder(true).SetTitle(" Crypto News ")
			tvBtc := tview.NewTextView().SetDynamicColors(true).SetText(renderCryptoBtcEtf(baseURL))
			tvBtc.SetBorder(true).SetTitle(" BTC ETF Tracker ")
			cryptoSubpanelViews[0], cryptoSubpanelViews[1], cryptoSubpanelViews[2], cryptoSubpanelViews[3] = tvTop, tvStable, tvNews, tvBtc
			topRow := tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(tvTop, 0, 1, false).
				AddItem(tvStable, 0, 1, false)
			botRow := tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(tvNews, 0, 1, false).
				AddItem(tvBtc, 0, 1, false)
			cryptoFlex := tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(topRow, 0, 1, false).
				AddItem(botRow, 0, 1, false)
			grid.AddItem(cryptoFlex, row, col, 1, 1, 0, 0, false)
			textViews[i] = nil
			continue
		}
		title := panelTitle(key)
		content := panelContent(baseURL, key)
		tv := tview.NewTextView().
			SetDynamicColors(true).
			SetText(content)
		tv.SetBorder(true).SetTitle(" " + title + " ")
		textViews[i] = tv
		grid.AddItem(tv, row, col, 1, 1, 0, 0, false)
	}

	// Footer row
	footer := tview.NewTextView().SetText(fmt.Sprintf(" Next in %ds · Q quit · Ctrl+C exit ", cycleSecs)).SetTextAlign(tview.AlignCenter)
	footer.SetBorder(false)
	grid.AddItem(footer, rows, 0, 1, cols, 0, 0, false)

	// Find which grid slot is the weather panel and which is crypto (own tickers)
	weatherPanelIndex := -1
	cryptoPanelIndex := -1
	for i := 0; i < n; i++ {
		if panels[i] == "weather" {
			weatherPanelIndex = i
		}
		if panels[i] == "crypto" {
			cryptoPanelIndex = i
		}
	}

	// Refresh all panels on a timer
	go func() {
		ticker := time.NewTicker(time.Duration(cycleSecs) * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			for i := 0; i < n; i++ {
				if i == weatherPanelIndex || i == cryptoPanelIndex {
					continue
				}
				key := panels[i]
				content := panelContent(baseURL, key)
				idx := i
				func(c string, j int) {
					app.QueueUpdateDraw(func() {
						textViews[j].SetText(c)
					})
				}(content, idx)
			}
			app.QueueUpdateDraw(func() {
				footer.SetText(fmt.Sprintf(" Next in %ds · Q quit · Ctrl+C exit ", cycleSecs))
			})
		}
	}()

	// Weather panel: refresh every 4s so continent cycling is visible (advances every 4s)
	if weatherPanelIndex >= 0 {
		go func() {
			wi := weatherPanelIndex
			ticker := time.NewTicker(4 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				content := panelContent(baseURL, "weather")
				func(c string) {
					app.QueueUpdateDraw(func() {
						textViews[wi].SetText(c)
					})
				}(content)
			}
		}()
	}

	// Crypto panel: 4 sub-panels with individual timers. Top Cryptos: 30 coins, 10 per page (1-10, 11-20, 21-30), 16s per page, live countdown in title.
	if cryptoPanelIndex >= 0 && cryptoSubpanelViews[0] != nil {
		vTop := cryptoSubpanelViews[0]
		vStable := cryptoSubpanelViews[1]
		vNews := cryptoSubpanelViews[2]
		vBtc := cryptoSubpanelViews[3]
		// Top Cryptos: 3 pages (1-10, 11-20, 21-30), 16s per page, title shows live countdown e.g. "Top 11-20 by mcap (16s)"
		go func() {
			rangeStarts := []int{1, 11, 21}
			pageIndex := 0
			secondsLeft := 16
			rangeLabel := func(start int) string {
				end := start + 9
				if end > 30 {
					end = 30
				}
				return fmt.Sprintf("%d-%d", start, end)
			}
			refreshContent := func() {
				start := rangeStarts[pageIndex]
				c := renderCryptoTop(baseURL, start)
				label := rangeLabel(start)
				titleTop := fmt.Sprintf(" Top %s by mcap (%ds) ", label, secondsLeft)
				app.QueueUpdateDraw(func() {
					vTop.SetTitle(titleTop)
					vTop.SetText(c)
				})
			}
			refreshContent()
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				secondsLeft--
				start := rangeStarts[pageIndex]
				label := rangeLabel(start)
				titleTop := fmt.Sprintf(" Top %s by mcap (%ds) ", label, secondsLeft)
				app.QueueUpdateDraw(func() {
					vTop.SetTitle(titleTop)
				})
				if secondsLeft <= 0 {
					pageIndex = (pageIndex + 1) % 3
					secondsLeft = 16
					refreshContent()
				}
			}
		}()
		// Stablecoins: refresh every 6s
		go func() {
			ticker := time.NewTicker(6 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				c := renderCryptoStablecoins(baseURL)
				app.QueueUpdateDraw(func() { vStable.SetText(c) })
			}
		}()
		// Crypto News: refresh every 6s
		go func() {
			ticker := time.NewTicker(6 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				c := renderCryptoNews(baseURL)
				app.QueueUpdateDraw(func() { vNews.SetText(c) })
			}
		}()
		// BTC ETF: refresh every 6s
		go func() {
			ticker := time.NewTicker(6 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				c := renderCryptoBtcEtf(baseURL)
				app.QueueUpdateDraw(func() { vBtc.SetText(c) })
			}
		}()
	}

	// Global key: Q to quit
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' || event.Rune() == 'Q' {
			app.Stop()
			return nil
		}
		return event
	})

	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetText("Pi Terminal World Monitor").SetTextAlign(tview.AlignCenter).SetBorder(false), 1, 0, false).
		AddItem(grid, 0, 1, true)

	if err := app.SetRoot(root, true).SetFocus(grid).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
