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
	Name        string `json:"name"`
	Temp        string `json:"temp"`
	TempHigh    string `json:"temp_high"`
	TempLow     string `json:"temp_low"`
	Conditions  string `json:"conditions"`
	WeatherCode int    `json:"weather_code"`
	Timezone    string `json:"timezone"`
}

type weatherContinent struct {
	Name      string       `json:"name"`
	Locations []weatherLoc `json:"locations"`
}

type weatherResp struct {
	Status     string             `json:"status"`
	Message    string             `json:"message"`
	Continents []weatherContinent `json:"continents"`
	Locations  []weatherLoc        `json:"locations"` // legacy flat list
}

type weatherNewsItem struct {
	Title       string `json:"title"`
	Link        string `json:"link"`
	PubDate     string `json:"pub_date"`
	Description string `json:"description"`
	Source      string `json:"source"`
}

type weatherNewsResp struct {
	Status string            `json:"status"`
	Source string            `json:"source"`
	Items  []weatherNewsItem `json:"items"`
}

type gsmRegion struct {
	Name     string   `json:"name"`
	Severity string   `json:"severity"`
	Events   []string `json:"events"`
}

type gsmLayer struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Icon      string   `json:"icon"`
	Active    bool     `json:"active"`
	Locations []string `json:"locations"`
}

type globalSituationMapResp struct {
	Status     string            `json:"status"`
	Source     string            `json:"source"`
	Defcon     int               `json:"defcon"`
	DefconPct  int               `json:"defcon_pct"`
	TimeWindow string            `json:"time_window"`
	UpdatedUTC string            `json:"updated_utc"`
	Summary    map[string][]string `json:"summary"`
	Layers     []gsmLayer        `json:"layers"`
	Regions    []gsmRegion       `json:"regions"`
	Message    string            `json:"message"`
}

// Crypto panel (4 sub-panels: top cryptos, stablecoins, crypto news, btc-etf)
type cryptoCoin struct {
	Rank        int      `json:"rank"`
	Symbol      string   `json:"symbol"`
	Name        string   `json:"name"`
	Price       float64  `json:"price"`
	Price1hPct  *float64 `json:"price_1h_pct"`
	Price24hPct *float64 `json:"price_24h_pct"`
	Price7dPct  *float64 `json:"price_7d_pct"`
}

type cryptoTopResp struct {
	Status string       `json:"status"`
	Range  string       `json:"range"`
	Coins  []cryptoCoin `json:"coins"`
}

type cryptoStablecoin struct {
	Symbol           string   `json:"symbol"`
	Name             string   `json:"name"`
	Price            float64  `json:"price"`
	PegStatus        string   `json:"peg_status"`
	DeviationPct     float64  `json:"deviation_pct"`
	MarketCapB       *float64 `json:"market_cap_b"`
	VolumeB          *float64 `json:"volume_b"`
	PriceChange24hPct *float64 `json:"price_change_24h_pct"`
}

type cryptoStablecoinsResp struct {
	Status      string             `json:"status"`
	StatusLabel string             `json:"status_label"`
	MarketCapB  *float64           `json:"market_cap_b"`
	VolumeB     *float64           `json:"volume_b"`
	Coins       []cryptoStablecoin `json:"coins"`
}

type cryptoGainersLosersEntry struct {
	Symbol        string   `json:"symbol"`
	Price         float64  `json:"price"`
	Change24hPct *float64 `json:"change_24h_pct"`
}

type cryptoGainersLosersResp struct {
	Status  string                     `json:"status"`
	Gainers []cryptoGainersLosersEntry `json:"gainers"`
	Losers  []cryptoGainersLosersEntry `json:"losers"`
}

// BTC ETF Tracker: header (Net Flow, Est. Flow, Total Vol, ETFs) + table rows
type cryptoBtcEtfEntry struct {
	Ticker     string  `json:"ticker"`
	Issuer     string  `json:"issuer"`
	EstFlowM   float64 `json:"est_flow_m"`
	VolumeM    float64 `json:"volume_m"`
	ChangePct  float64 `json:"change_pct"`
}

type cryptoBtcEtfResp struct {
	Status       string              `json:"status"`
	Source       string              `json:"source"`
	NetFlowLabel string              `json:"net_flow_label"`
	EstFlowM     float64             `json:"est_flow_m"`
	TotalVolM    float64             `json:"total_vol_m"`
	EtfsUp       int                  `json:"etfs_up"`
	EtfsDown     int                  `json:"etfs_down"`
	Etfs         []cryptoBtcEtfEntry `json:"etfs"`
}

type cryptoNewsItem struct {
	Title       string `json:"title"`
	Link        string `json:"link"`
	PubDate     string `json:"pub_date"`
	Description string `json:"description"`
	Source      string `json:"source"`
}

// News panel: 8 feeds (World, US, Europe, Middle East, Africa, Asia-Pacific, Energy, Government)
type newsItem struct {
	Title       string `json:"title"`
	Link        string `json:"link"`
	PubDate     string `json:"pub_date"`
	Description string `json:"description"`
	Source      string `json:"source"`
}

type newsFeed struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	NewCount int        `json:"new_count"`
	Items    []newsItem `json:"items"`
}

type newsResp struct {
	Status string     `json:"status"`
	Source string     `json:"source"`
	Feeds  []newsFeed `json:"feeds"`
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
		return []string{"crypto", "weather", "news", "world-clock"}, nil
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
		localTime := ""
		if loc.Timezone != "" {
			if tz, err := time.LoadLocation(loc.Timezone); err == nil {
				localTime = time.Now().In(tz).Format("15:04")
			}
		}
		if localTime != "" {
			b.WriteString(fmt.Sprintf("  %s %s %s%s°[-] (%s°/ %s°) %s %s\n", icon, loc.Name, tag, loc.Temp, lo, hi, loc.Conditions, localTime))
		} else {
			b.WriteString(fmt.Sprintf("  %s %s %s%s°[-] (%s°/ %s°) %s\n", icon, loc.Name, tag, loc.Temp, lo, hi, loc.Conditions))
		}
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

// renderCryptoTopWithRange fetches and renders the top-cryptos panel; perPage sets lines per page (resolution-aware). Returns content and range label.
func renderCryptoTopWithRange(baseURL string, rangeStart, perPage int) (content string, rangeLabel string) {
	if perPage < 5 {
		perPage = 5
	}
	if perPage > 25 {
		perPage = 25
	}
	rangeLabel = fmt.Sprintf("%d-%d", rangeStart, rangeStart+perPage-1)
	var d cryptoTopResp
	path := fmt.Sprintf("/panels/crypto/top?range_start=%d&per_page=%d", rangeStart, perPage)
	if err := fetchJSON(baseURL, path, &d); err != nil {
		return "No data", rangeLabel
	}
	if d.Status != "ok" || len(d.Coins) == 0 {
		return "No data", rangeLabel
	}
	if d.Range != "" {
		rangeLabel = d.Range
	}
	var b strings.Builder
	for _, c := range d.Coins {
		p1h := "—"
		if c.Price1hPct != nil {
			p1h = fmt.Sprintf("%+.2f%%", *c.Price1hPct)
		}
		p24 := "—"
		if c.Price24hPct != nil {
			p24 = fmt.Sprintf("%+.2f%%", *c.Price24hPct)
		}
		p7d := "—"
		if c.Price7dPct != nil {
			p7d = fmt.Sprintf("%+.2f%%", *c.Price7dPct)
		}
		tag1h := pctColor(c.Price1hPct)
		tag24 := pctColor(c.Price24hPct)
		tag7d := pctColor(c.Price7dPct)
		b.WriteString(fmt.Sprintf("  %2d %s %s 1h %s%s[-] 24h %s%s[-] 7d %s%s[-]\n", c.Rank, c.Symbol, fmtPrice(c.Price), tag1h, p1h, tag24, p24, tag7d, p7d))
	}
	return strings.TrimSuffix(b.String(), "\n"), rangeLabel
}

func renderCryptoTop(baseURL string, rangeStart int) string {
	content, _ := renderCryptoTopWithRange(baseURL, rangeStart, 11)
	return content
}

// formatStableVol formats volume for stablecoins: 54.7B or 132M
func formatStableVol(b *float64) string {
	if b == nil || *b == 0 {
		return "—"
	}
	if *b >= 1 {
		return fmt.Sprintf("$%.1fB", *b)
	}
	return fmt.Sprintf("$%.0fM", *b*1000)
}

// renderCryptoStablecoinsPageFromData renders one page: tile = status then MCap|Vol; tickers = one line each (no mcap/vol per coin).
func renderCryptoStablecoinsPageFromData(d *cryptoStablecoinsResp, pageStart, perPage int) string {
	if d == nil || len(d.Coins) == 0 {
		return "No data"
	}
	n := len(d.Coins)
	if pageStart >= n {
		pageStart = 0
	}
	end := pageStart + perPage
	if end > n {
		end = n
	}
	page := d.Coins[pageStart:end]
	var b strings.Builder
	statusTag := "[green]"
	if d.StatusLabel != "Healthy" {
		statusTag = "[yellow]"
	}
	b.WriteString(fmt.Sprintf(" %s%s[-]\n", statusTag, d.StatusLabel))
	if d.MarketCapB != nil && d.VolumeB != nil {
		b.WriteString(fmt.Sprintf(" MCap: $%.1fB | Vol: $%.1fB\n", *d.MarketCapB, *d.VolumeB))
	}
	b.WriteString("\n")
	for _, c := range page {
		pegTag := "[green]"
		if c.PegStatus != "ON PEG" {
			pegTag = "[red]"
		}
		b.WriteString(fmt.Sprintf("  %-6s  $%.2f  %s%s[-] %.2f%%\n", c.Symbol, c.Price, pegTag, c.PegStatus, c.DeviationPct))
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func renderCryptoStablecoinsPage(baseURL string, pageStart, perPage int) string {
	var d cryptoStablecoinsResp
	if err := fetchJSON(baseURL, "/panels/crypto/stablecoins", &d); err != nil {
		return "No data"
	}
	if d.Status != "ok" {
		return "No data"
	}
	return renderCryptoStablecoinsPageFromData(&d, pageStart, perPage)
}

func renderCryptoStablecoins(baseURL string) string {
	return renderCryptoStablecoinsPage(baseURL, 0, 6)
}

const gainersLosersPerPage = 12

func renderCryptoGainersLosers(baseURL string, showGainers bool, pageStart, perPage int) string {
	var d cryptoGainersLosersResp
	if err := fetchJSON(baseURL, "/panels/crypto/gainers-losers", &d); err != nil {
		return "No data"
	}
	if d.Status != "ok" {
		return "No data"
	}
	list := d.Losers
	tag := "[red]"
	title := "Losers"
	if showGainers {
		list = d.Gainers
		tag = "[green]"
		title = "Gainers"
	}
	n := len(list)
	if pageStart >= n {
		pageStart = 0
	}
	end := pageStart + perPage
	if end > n {
		end = n
	}
	page := list[pageStart:end]
	var b strings.Builder
	b.WriteString(fmt.Sprintf(" [::b]Crypto %s[-]\n\n", title))
	for _, e := range page {
		priceStr := fmt.Sprintf("%.4f", e.Price)
		if e.Price >= 1000 {
			priceStr = fmt.Sprintf("%.2f", e.Price)
		} else if e.Price >= 1 {
			priceStr = fmt.Sprintf("%.2f", e.Price)
		}
		chgStr := "—"
		if e.Change24hPct != nil {
			chgStr = fmt.Sprintf("%+.2f%%", *e.Change24hPct)
		}
		b.WriteString(fmt.Sprintf("  %s%-8s[-] $%-10s  %s%s[-]\n", tag, e.Symbol, priceStr, tag, chgStr))
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// formatBtcEtfFlow formats est flow for display: -$220.2M or -$364K
func formatBtcEtfFlow(m float64) string {
	abs := m
	if abs < 0 {
		abs = -abs
	}
	if abs >= 1 {
		return fmt.Sprintf("-$%.1fM", abs)
	}
	return fmt.Sprintf("-$%.0fK", abs*1000)
}

// formatBtcEtfVol formats volume for display: 57.0M or 189K
func formatBtcEtfVol(m float64) string {
	if m >= 1 {
		return fmt.Sprintf("%.1fM", m)
	}
	return fmt.Sprintf("%.0fK", m*1000)
}

// BTC ETF column widths for alignment (header and data use same positions)
const (
	btcColTicker = 6
	btcColIssuer = 15
	btcColFlow   = 11
	btcColVol    = 8
	btcColChange = 8
)

// renderCryptoBtcEtfAll renders the full BTC ETF Tracker: header + all ETFs at once; fixed-width columns for alignment.
func renderCryptoBtcEtfAll(baseURL string) string {
	var d cryptoBtcEtfResp
	if err := fetchJSON(baseURL, "/panels/crypto/btc-etf", &d); err != nil {
		return "No data"
	}
	if d.Status != "ok" {
		return "No data"
	}
	var b strings.Builder
	netTag := "[red]"
	if d.NetFlowLabel == "NET INFLOW" {
		netTag = "[green]"
	}
	firstColWidth := btcColTicker + 1 + btcColIssuer
	b.WriteString(fmt.Sprintf(" %s%-*s[-] %*s %*s %*s\n",
		netTag, firstColWidth, d.NetFlowLabel,
		btcColFlow, fmt.Sprintf("$%.1fM", d.EstFlowM),
		btcColVol, fmt.Sprintf("%.1fM", d.TotalVolM),
		btcColChange, fmt.Sprintf("%d↑ %d↓", d.EtfsUp, d.EtfsDown)))
	b.WriteString(fmt.Sprintf(" [gray]%-*s %-*s %*s %*s %*s[-]\n",
		btcColTicker, "TICKER", btcColIssuer, "ISSUER", btcColFlow, "EST. FLOW", btcColVol, "VOLUME", btcColChange, "CHANGE"))
	for _, e := range d.Etfs {
		flowStr := formatBtcEtfFlow(e.EstFlowM)
		volStr := formatBtcEtfVol(e.VolumeM)
		chStr := fmt.Sprintf("%+.2f%%", e.ChangePct)
		chTag := "[red]"
		if e.ChangePct >= 0 {
			chTag = "[green]"
		}
		b.WriteString(fmt.Sprintf(" [white]%-*s[-] %-*s %s%*s[-] %*s %s%*s[-]\n",
			btcColTicker, e.Ticker, btcColIssuer, e.Issuer,
			"[red]", btcColFlow, flowStr, btcColVol, volStr,
			chTag, btcColChange, chStr))
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func renderCryptoBtcEtf(baseURL string) string {
	return renderCryptoBtcEtfAll(baseURL)
}

// wrapLines splits text into lines of at most width runes (word boundaries when possible).
func wrapLines(s string, width int) []string {
	if width <= 0 {
		width = 60
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var lines []string
	runes := []rune(s)
	for len(runes) > 0 {
		if len(runes) <= width {
			lines = append(lines, string(runes))
			break
		}
		chunk := runes[:width]
		lastSpace := -1
		for i := len(chunk) - 1; i >= 0; i-- {
			if chunk[i] == ' ' || chunk[i] == '\n' {
				lastSpace = i
				break
			}
		}
		if lastSpace > 0 {
			lines = append(lines, string(runes[:lastSpace]))
			runes = runes[lastSpace+1:]
		} else {
			lines = append(lines, string(runes[:width]))
			runes = runes[width:]
		}
		runes = []rune(strings.TrimLeft(string(runes), " \n\t"))
	}
	return lines
}

// fetchCryptoNewsItems returns the current news items from the API (for cycling in the Crypto News panel).
func fetchCryptoNewsItems(baseURL string) ([]cryptoNewsItem, error) {
	var d cryptoNewsResp
	if err := fetchJSON(baseURL, "/panels/crypto/news", &d); err != nil {
		return nil, err
	}
	if d.Status != "ok" || len(d.Items) == 0 {
		return nil, nil
	}
	return d.Items, nil
}

// renderCryptoNewsOneArticle appends one article (headline + blurb); blurb in italic. Wrap width = width-2 so indent + line fits in panel.
func renderCryptoNewsOneArticle(b *strings.Builder, it *cryptoNewsItem, width, maxLines int) int {
	if maxLines <= 0 {
		return 0
	}
	indent := "  "
	contentWidth := width - 2
	if contentWidth < 12 {
		contentWidth = 12
	}
	// Headline: title only (source on its own line so it's not cut off)
	titleLines := wrapLines(strings.TrimSpace(it.Title), contentWidth)
	maxTitleLines := 2
	if maxTitleLines > maxLines-3 {
		maxTitleLines = maxLines - 3 // reserve line for source + blank
	}
	linesUsed := 0
	for i, w := range titleLines {
		if i >= maxTitleLines {
			break
		}
		b.WriteString(indent)
		b.WriteString(strings.TrimSpace(w))
		b.WriteString("\n")
		linesUsed++
	}
	// Source on its own line
	if it.Source != "" {
		b.WriteString(indent)
		b.WriteString("(" + it.Source + ")\n")
		linesUsed++
	}
	b.WriteString("\n")
	linesUsed++
	if linesUsed >= maxLines {
		return linesUsed
	}
	blurb := strings.TrimSpace(it.Description)
	if blurb == "" {
		blurb = "—"
	}
	wrapped := wrapLines(blurb, contentWidth)
	remaining := maxLines - linesUsed
	for i, w := range wrapped {
		if i >= remaining {
			break
		}
		b.WriteString(indent)
		b.WriteString("[::i]")
		b.WriteString(strings.TrimSpace(w))
		b.WriteString("[::-]\n")
		linesUsed++
	}
	return linesUsed
}

// renderCryptoNewsTwoItems renders two headlines + descriptions per cycle with a separator between them.
// Headlines wrap to width (no truncation). Advances by 2 items per page.
func renderCryptoNewsTwoItems(items []cryptoNewsItem, pageIndex, width, contentHeight int) string {
	if contentHeight <= 0 {
		contentHeight = 20
	}
	if width <= 0 {
		width = 60
	}
	if len(items) == 0 {
		return "No headlines"
	}
	n := len(items)
	idx0 := pageIndex % n
	if idx0 < 0 {
		idx0 += n
	}
	idx1 := (pageIndex + 1) % n
	sepLines := 2 // em dash line + one blank below
	half := (contentHeight - sepLines) / 2
	if half < 3 {
		half = 3
	}
	var b strings.Builder
	lines1 := renderCryptoNewsOneArticle(&b, &items[idx0], width, half)
	b.WriteString("\n  —\n") // em dash, one blank below
	lines2 := renderCryptoNewsOneArticle(&b, &items[idx1], width, contentHeight-half-sepLines)
	_, _ = lines1, lines2
	return strings.TrimSuffix(b.String(), "\n")
}

func renderCryptoNews(baseURL string) string {
	items, _ := fetchCryptoNewsItems(baseURL)
	return renderCryptoNewsTwoItems(items, 0, 60, 24)
}

// News panel timers: first (top-left) 25s, then 30s, 35s, ... 60s so they refresh in order one at a time
func newsPanelCycleSecs(panelIndex int) int {
	return 25 + panelIndex*5
}

func fetchNewsFeeds(baseURL string) ([]newsFeed, error) {
	var d newsResp
	if err := fetchJSON(baseURL, "/panels/news", &d); err != nil {
		return nil, err
	}
	if d.Status != "ok" || len(d.Feeds) == 0 {
		return nil, nil
	}
	return d.Feeds, nil
}

func fetchWeatherNewsItems(baseURL string) ([]weatherNewsItem, error) {
	var d weatherNewsResp
	if err := fetchJSON(baseURL, "/panels/weather/news", &d); err != nil {
		return nil, err
	}
	if d.Status != "ok" {
		return nil, nil
	}
	return d.Items, nil
}

func renderWeatherNewsOneItem(it *weatherNewsItem, width, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	contentWidth := width - 2
	if contentWidth < 12 {
		contentWidth = 12
	}
	var b strings.Builder
	indent := "  "
	titleLines := wrapLines(strings.TrimSpace(it.Title), contentWidth)
	maxTitleLines := 2
	if maxTitleLines > maxLines-3 {
		maxTitleLines = maxLines - 3
	}
	linesUsed := 0
	for i, w := range titleLines {
		if i >= maxTitleLines {
			break
		}
		b.WriteString(indent)
		b.WriteString(strings.TrimSpace(w))
		b.WriteString("\n")
		linesUsed++
	}
	if it.Source != "" {
		b.WriteString(indent)
		b.WriteString("(" + it.Source + ")\n")
		linesUsed++
	}
	b.WriteString("\n")
	linesUsed++
	if linesUsed >= maxLines {
		return strings.TrimSuffix(b.String(), "\n")
	}
	blurb := strings.TrimSpace(it.Description)
	if blurb == "" {
		blurb = "—"
	}
	for _, w := range wrapLines(blurb, contentWidth) {
		if linesUsed >= maxLines {
			break
		}
		b.WriteString(indent)
		b.WriteString("[::i]")
		b.WriteString(strings.TrimSpace(w))
		b.WriteString("[::-]\n")
		linesUsed++
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// renderNewsOneArticle formats one news item as headline + blurb; description in italic, aligned for readability.
func renderNewsOneArticle(it *newsItem, width, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	// Use consistent wrap width (indent is 2 spaces; content width = width - 2)
	contentWidth := width - 2
	if contentWidth < 12 {
		contentWidth = 12
	}
	var b strings.Builder
	indent := "  "
	// Headline: title only; source on its own line so it's not cut off
	titleLines := wrapLines(strings.TrimSpace(it.Title), contentWidth)
	maxTitleLines := 2
	if maxTitleLines > maxLines-3 {
		maxTitleLines = maxLines - 3
	}
	linesUsed := 0
	for i, w := range titleLines {
		if i >= maxTitleLines {
			break
		}
		b.WriteString(indent)
		b.WriteString(strings.TrimSpace(w))
		b.WriteString("\n")
		linesUsed++
	}
	if it.Source != "" {
		b.WriteString(indent)
		b.WriteString("(" + it.Source + ")\n")
		linesUsed++
	}
	b.WriteString("\n")
	linesUsed++
	if linesUsed >= maxLines {
		return strings.TrimSuffix(b.String(), "\n")
	}
	blurb := strings.TrimSpace(it.Description)
	if blurb == "" {
		blurb = "—"
	}
	wrapped := wrapLines(blurb, contentWidth)
	remaining := maxLines - linesUsed
	for i, w := range wrapped {
		if i >= remaining {
			break
		}
		b.WriteString(indent)
		// Italic for article description (tview: [::i]...[::-])
		b.WriteString("[::i]")
		b.WriteString(strings.TrimSpace(w))
		b.WriteString("[::-]\n")
		linesUsed++
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// renderCrypto is used only when crypto is a single panel (legacy); prefer crypto 3-subpanel layout.
func renderCrypto(baseURL string) string {
	return renderCryptoTop(baseURL, 1)
}

func gsmSeverityTag(severity string) string {
	switch strings.ToLower(severity) {
	case "critical", "high":
		return "[red]"
	case "elevated":
		return "[yellow]"
	case "monitoring":
		return "[cyan]"
	case "normal":
		return "[green]"
	default:
		return "[white]"
	}
}

func buildGsmHeader(d *globalSituationMapResp) string {
	if d.TimeWindow == "" {
		d.TimeWindow = "7d"
	}
	s := fmt.Sprintf("%s · DEFCON %d", d.TimeWindow, d.Defcon)
	if d.DefconPct > 0 {
		s += fmt.Sprintf(" %d%%", d.DefconPct)
	}
	if d.UpdatedUTC != "" {
		// show time only, e.g. "00:35 UTC"
		if len(d.UpdatedUTC) >= 16 {
			s += " · " + d.UpdatedUTC[11:16] + " UTC"
		}
	}
	return s
}

func buildGsmAlerts(d *globalSituationMapResp) string {
	if len(d.Summary) == 0 {
		return ""
	}
	levelLabels := map[string]string{"high": "High", "elevated": "Elevated", "monitoring": "Monitoring"}
	var parts []string
	for _, level := range []string{"high", "elevated", "monitoring"} {
		locs := d.Summary[level]
		if len(locs) == 0 {
			continue
		}
		label := levelLabels[level]
		if label == "" {
			label = level
		}
		tag := gsmSeverityTag(level)
		parts = append(parts, fmt.Sprintf("%s%s[-]: %s", tag, label, strings.Join(locs, ", ")))
	}
	return strings.Join(parts, " | ")
}

func buildGsmLayersRegions(d *globalSituationMapResp) string {
	var b strings.Builder
	if len(d.Layers) > 0 {
		b.WriteString("[yellow]Layers[-]\n")
		for _, l := range d.Layers {
			if !l.Active || len(l.Locations) == 0 {
				continue
			}
			b.WriteString(fmt.Sprintf("  %s %s: %s\n", l.Icon, l.Name, strings.Join(l.Locations, ", ")))
		}
	}
	b.WriteString("[yellow]Regions[-] ")
	for i, r := range d.Regions {
		if i > 0 {
			b.WriteString(" | ")
		}
		tag := gsmSeverityTag(r.Severity)
		b.WriteString(fmt.Sprintf("%s%s[-] %s", tag, r.Severity, r.Name))
		if len(r.Events) > 0 {
			b.WriteString(" (" + strings.Join(r.Events, ", ") + ")")
		}
	}
	return strings.TrimSpace(b.String())
}

func fetchAndBuildGsm(baseURL string) (header, alerts, body string, ok bool) {
	var d globalSituationMapResp
	if err := fetchJSON(baseURL, "/panels/global-situation-map", &d); err != nil {
		return "—", "No data", "", false
	}
	if d.Status != "ok" && d.Status != "" {
		return "—", d.Message, "", false
	}
	return buildGsmHeader(&d), buildGsmAlerts(&d), buildGsmLayersRegions(&d), true
}

func renderGlobalSituationMap(baseURL string) string {
	header, alerts, body := "", "", ""
	header, alerts, body, _ = fetchAndBuildGsm(baseURL)
	var b strings.Builder
	if header != "" && header != "—" {
		b.WriteString(header)
		b.WriteString("\n")
	}
	if alerts != "" {
		b.WriteString(alerts)
		b.WriteString("\n")
	}
	if body != "" {
		b.WriteString(body)
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
	case "news":
		return "News"
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
	case "news":
		feeds, _ := fetchNewsFeeds(baseURL)
		if len(feeds) == 0 || len(feeds[0].Items) == 0 {
			return "No news feeds"
		}
		return renderNewsOneArticle(&feeds[0].Items[0], 50, 12)
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

	// One TextView per cell (or Flex for crypto, news, or weather+world-clock split)
	textViews := make([]*tview.TextView, n)
	var cryptoSubpanelViews [5]*tview.TextView
	var newsSubpanelViews [8]*tview.TextView
	var weatherWatchView, weatherNewsView *tview.TextView
	var worldClockPanelIndex int = -1
	for i := 0; i < n; i++ {
		key := panels[i]
		row, col := i/cols, i%cols
		if key == "weather" {
			// Empty panel (weather moved down to world-clock slot)
			tv := tview.NewTextView().SetDynamicColors(true).SetText("")
			tv.SetBorder(true).SetTitle(" ")
			textViews[i] = tv
			grid.AddItem(tv, row, col, 1, 1, 0, 0, false)
			continue
		}
		if key == "world-clock" {
			worldClockPanelIndex = i
			weatherWatchView = tview.NewTextView().SetDynamicColors(true)
			weatherWatchView.SetBorder(true).SetTitle(" Weather Watch ")
			weatherWatchView.SetText(renderWeather(baseURL))
			weatherNewsView = tview.NewTextView().SetDynamicColors(true)
			weatherNewsView.SetBorder(true).SetTitle(" Weather News (25s) ")
			wnItems, _ := fetchWeatherNewsItems(baseURL)
			if len(wnItems) > 0 {
				weatherNewsView.SetText(renderWeatherNewsOneItem(&wnItems[0], 40, 20))
			} else {
				weatherNewsView.SetText("  No headlines")
			}
			weatherFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(weatherWatchView, 0, 1, false).
				AddItem(weatherNewsView, 0, 1, false)
			weatherFlex.SetBorder(false)
			grid.AddItem(weatherFlex, row, col, 1, 1, 0, 0, false)
			textViews[i] = nil
			continue
		}
		if key == "news" {
			feeds, _ := fetchNewsFeeds(baseURL)
			for j := 0; j < 8; j++ {
				tv := tview.NewTextView().SetDynamicColors(true)
				tv.SetBorder(true)
				title := " News "
				body := "No headlines"
				secLeft := newsPanelCycleSecs(j) // at t=0 each panel shows its full cycle (25, 30, ... 60)
				if j < len(feeds) {
					f := &feeds[j]
					newStr := ""
					if f.NewCount > 0 {
						newStr = fmt.Sprintf(" %d NEW ", f.NewCount)
					}
					tit := strings.TrimSpace(f.Name)
					if newStr != "" {
						tit += " " + strings.TrimSpace(newStr)
					}
					title = " " + tit + " (" + strconv.Itoa(secLeft) + "s) "
					if len(f.Items) > 0 {
						body = renderNewsOneArticle(&f.Items[0], 50, 20)
					}
				}
				tv.SetTitle(title)
				tv.SetText(body)
				newsSubpanelViews[j] = tv
			}
			topRow := tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(newsSubpanelViews[0], 0, 1, false).
				AddItem(newsSubpanelViews[1], 0, 1, false).
				AddItem(newsSubpanelViews[2], 0, 1, false).
				AddItem(newsSubpanelViews[3], 0, 1, false)
			botRow := tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(newsSubpanelViews[4], 0, 1, false).
				AddItem(newsSubpanelViews[5], 0, 1, false).
				AddItem(newsSubpanelViews[6], 0, 1, false).
				AddItem(newsSubpanelViews[7], 0, 1, false)
			newsFlex := tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(topRow, 0, 1, false).
				AddItem(botRow, 0, 1, false)
			newsFlex.SetBorder(true).SetTitle(" News ")
			grid.AddItem(newsFlex, row, col, 1, 1, 0, 0, false)
			textViews[i] = nil
			continue
		}
		if key == "crypto" {
			// Crypto: Top | (Stablecoins | Gainers/Losers); Crypto News | BTC ETF
			tvTop := tview.NewTextView().SetDynamicColors(true).SetText(renderCryptoTop(baseURL, 1))
			tvTop.SetBorder(true).SetTitle(" Top cryptos by mcap (8s) ")
			tvStable := tview.NewTextView().SetDynamicColors(true).SetText(renderCryptoStablecoins(baseURL))
			tvStable.SetBorder(true).SetTitle(" Stablecoins ")
			tvGainersLosers := tview.NewTextView().SetDynamicColors(true).SetText(renderCryptoGainersLosers(baseURL, true, 0, gainersLosersPerPage))
			tvGainersLosers.SetBorder(true).SetTitle(" Crypto gainers (10s) ")
			tvNews := tview.NewTextView().SetDynamicColors(true).SetText(renderCryptoNews(baseURL))
			tvNews.SetBorder(true).SetTitle(" Crypto News ")
			tvBtc := tview.NewTextView().SetDynamicColors(true).SetText(renderCryptoBtcEtfAll(baseURL))
			tvBtc.SetBorder(true).SetTitle(" BTC ETF Tracker (6s) ")
			cryptoSubpanelViews[0], cryptoSubpanelViews[1], cryptoSubpanelViews[2], cryptoSubpanelViews[3], cryptoSubpanelViews[4] = tvTop, tvStable, tvGainersLosers, tvNews, tvBtc
			stableGainersFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(tvStable, 0, 1, false).
				AddItem(tvGainersLosers, 0, 1, false)
			topRow := tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(tvTop, 0, 1, false).
				AddItem(stableGainersFlex, 0, 1, false)
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

	// Find which grid slot is weather, crypto, or news (each has its own refresh)
	weatherPanelIndex := -1
	cryptoPanelIndex := -1
	newsPanelIndex := -1
	for i := 0; i < n; i++ {
		if panels[i] == "weather" {
			weatherPanelIndex = i
		}
		if panels[i] == "crypto" {
			cryptoPanelIndex = i
		}
		if panels[i] == "news" {
			newsPanelIndex = i
		}
	}

	// Refresh all panels on a timer
	go func() {
		ticker := time.NewTicker(time.Duration(cycleSecs) * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			for i := 0; i < n; i++ {
				if i == weatherPanelIndex || i == cryptoPanelIndex || i == newsPanelIndex || i == worldClockPanelIndex {
					continue
				}
				if textViews[i] == nil {
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

	// Weather Watch (in world-clock slot): refresh every 4s so continent cycling is visible
	if weatherWatchView != nil {
		go func() {
			ticker := time.NewTicker(4 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				content := renderWeather(baseURL)
				app.QueueUpdateDraw(func() {
					weatherWatchView.SetText(content)
				})
			}
		}()
	}

	// Weather News (in world-clock slot): 25s cycle, 30s refresh, timer in title
	if weatherNewsView != nil {
		go func() {
			items, _ := fetchWeatherNewsItems(baseURL)
			itemIndex := 0
			tickCount := 0
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				tickCount++
				if tickCount%30 == 0 {
					if newItems, err := fetchWeatherNewsItems(baseURL); err == nil && len(newItems) > 0 {
						items = newItems
						itemIndex = 0
					}
				}
				if tickCount > 0 && tickCount%25 == 0 && len(items) > 0 {
					itemIndex = (itemIndex + 1) % len(items)
				}
				secLeft := 25 - (tickCount % 25)
				if secLeft == 25 {
					secLeft = 25
				}
				title := fmt.Sprintf(" Weather News (%ds) ", secLeft)
				var body string
				if len(items) > 0 {
					body = renderWeatherNewsOneItem(&items[itemIndex], 40, 20)
				} else {
					body = "  No headlines"
				}
				app.QueueUpdateDraw(func() {
					weatherNewsView.SetTitle(title)
					weatherNewsView.SetText(body)
				})
			}
		}()
	}

	// News panel: 8 sub-panels, 25s per panel; each panel offset by 5s (top-left to right); refresh feed data every 30s
	if newsPanelIndex >= 0 && newsSubpanelViews[0] != nil {
		go func() {
			feeds, _ := fetchNewsFeeds(baseURL)
			itemIndices := make([]int, 8)
			tickCount := 0
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				tickCount++
				if tickCount%30 == 0 {
					if newFeeds, err := fetchNewsFeeds(baseURL); err == nil && len(newFeeds) > 0 {
						feeds = newFeeds
					}
				}
				// Per-panel: cycle 25s, 30s, ... 60s; advance when tickCount is a multiple of that cycle
				for j := 0; j < 8; j++ {
					cycle := newsPanelCycleSecs(j)
					if tickCount > 0 && tickCount%cycle == 0 {
						if j < len(feeds) && len(feeds[j].Items) > 0 {
							itemIndices[j] = (itemIndices[j] + 1) % len(feeds[j].Items)
						}
					}
				}
				app.QueueUpdateDraw(func() {
					for j := 0; j < 8; j++ {
						v := newsSubpanelViews[j]
						if v == nil {
							continue
						}
						_, _, w, h := v.GetRect()
						if w < 20 {
							w = 50
						}
						if h < 5 {
							h = 20
						}
						cycle := newsPanelCycleSecs(j)
						secLeft := cycle - (tickCount % cycle)
						if secLeft == 0 {
							secLeft = cycle
						}
						newStr := ""
						body := "No headlines"
						if j < len(feeds) {
							f := &feeds[j]
							if f.NewCount > 0 {
								newStr = fmt.Sprintf(" %d NEW ", f.NewCount)
							}
							// Title: single spaces throughout so alignment is consistent
							title := strings.TrimSpace(f.Name)
							if newStr != "" {
								title += " " + strings.TrimSpace(newStr)
							}
							v.SetTitle(" " + title + " (" + strconv.Itoa(secLeft) + "s) ")
							if len(f.Items) > 0 {
								idx := itemIndices[j] % len(f.Items)
								body = renderNewsOneArticle(&f.Items[idx], w, h-2)
							}
						}
						v.SetText(body)
					}
				})
			}
		}()
	}

	// Crypto panel: 5 sub-panels. Top | (Stablecoins | Gainers/Losers); News | BTC ETF.
	if cryptoPanelIndex >= 0 && cryptoSubpanelViews[0] != nil {
		vTop := cryptoSubpanelViews[0]
		vStable := cryptoSubpanelViews[1]
		vGainersLosers := cryptoSubpanelViews[2]
		vNews := cryptoSubpanelViews[3]
		vBtc := cryptoSubpanelViews[4]
		// Top Cryptos: 56 coins; 8s per page; lines per page from panel height (resolution-aware)
		go func() {
			const topCoinsCount = 56
			perPage := 11
			rangeStarts := []int{1, 12, 23, 34, 45}
			pageIndex := 0
			secondsLeft := 8
			refreshContent := func() {
				ch := make(chan int, 1)
				app.QueueUpdateDraw(func() {
					_, _, _, h := vTop.GetRect()
					if h > 3 {
						ch <- h - 2
					} else {
						ch <- 11
					}
					close(ch)
				})
				received := 11
				if v, ok := <-ch; ok {
					received = v
				}
				if received >= 5 && received <= 25 {
					perPage = received
					numPages := (topCoinsCount + perPage - 1) / perPage
					rangeStarts = make([]int, numPages)
					for i := 0; i < numPages; i++ {
						rangeStarts[i] = i*perPage + 1
					}
				}
				if pageIndex >= len(rangeStarts) {
					pageIndex = 0
				}
				start := rangeStarts[pageIndex]
				c, _ := renderCryptoTopWithRange(baseURL, start, perPage)
				titleTop := fmt.Sprintf(" Top cryptos by mcap (%ds) ", secondsLeft)
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
				if secondsLeft <= 0 {
					pageIndex = (pageIndex + 1) % len(rangeStarts)
					secondsLeft = 8
					refreshContent()
					continue
				}
				titleTop := fmt.Sprintf(" Top cryptos by mcap (%ds) ", secondsLeft)
				app.QueueUpdateDraw(func() {
					vTop.SetTitle(titleTop)
				})
			}
		}()
		// Stablecoins: cycle pages like Top Cryptos (8s per page), ticker only, 2 decimals, one line per coin
		go func() {
			stableSecs := 8
			secondsLeft := stableSecs
			pageIndex := 0
			perPage := 4
			var lastData *cryptoStablecoinsResp
			refreshStable := func() {
				ch := make(chan int, 1)
				app.QueueUpdateDraw(func() {
					_, _, _, h := vStable.GetRect()
					if h > 6 {
						// tile 2 lines + 1 blank + 1 line per coin
						p := h - 2 - 3
						if p >= 1 && p <= 20 {
							ch <- p
						} else {
							ch <- perPage
						}
					} else {
						ch <- perPage
					}
					close(ch)
				})
				if v, ok := <-ch; ok && v >= 1 && v <= 20 {
					perPage = v
				}
				var d cryptoStablecoinsResp
				if err := fetchJSON(baseURL, "/panels/crypto/stablecoins", &d); err == nil && d.Status == "ok" && len(d.Coins) > 0 {
					lastData = &d
					n := len(d.Coins)
					numPages := (n + perPage - 1) / perPage
					if numPages < 1 {
						numPages = 1
					}
					pageIndex = pageIndex % numPages
					pageStart := pageIndex * perPage
					c := renderCryptoStablecoinsPageFromData(lastData, pageStart, perPage)
					title := fmt.Sprintf(" Stablecoins (%ds) ", secondsLeft)
					app.QueueUpdateDraw(func() {
						vStable.SetTitle(title)
						vStable.SetText(c)
					})
					pageIndex = (pageIndex + 1) % numPages
				} else if lastData != nil {
					n := len(lastData.Coins)
					numPages := (n + perPage - 1) / perPage
					if numPages < 1 {
						numPages = 1
					}
					pageIndex = pageIndex % numPages
					pageStart := pageIndex * perPage
					c := renderCryptoStablecoinsPageFromData(lastData, pageStart, perPage)
					app.QueueUpdateDraw(func() {
						vStable.SetTitle(fmt.Sprintf(" Stablecoins (%ds) ", secondsLeft))
						vStable.SetText(c)
					})
					pageIndex = (pageIndex + 1) % numPages
				}
			}
			refreshStable()
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				secondsLeft--
				if secondsLeft <= 0 {
					secondsLeft = stableSecs
					refreshStable()
					continue
				}
				app.QueueUpdateDraw(func() {
					vStable.SetTitle(fmt.Sprintf(" Stablecoins (%ds) ", secondsLeft))
				})
			}
		}()
		// Gainers/Losers: 12 per page; cycle gainers p0 → gainers p1 → losers p0 → losers p1 every 10s; show change % next to price
		go func() {
			gainersSecs := 10
			secondsLeft := gainersSecs
			phase := 0 // 0 = gainers 0-11, 1 = gainers 12-23, 2 = losers 0-11, 3 = losers 12-23
			refreshGL := func() {
				showGainers := phase < 2
				pageStart := 0
				if phase == 1 || phase == 3 {
					pageStart = gainersLosersPerPage
				}
				c := renderCryptoGainersLosers(baseURL, showGainers, pageStart, gainersLosersPerPage)
				label := "gainers"
				if !showGainers {
					label = "losers"
				}
				title := fmt.Sprintf(" Crypto %s (%ds) ", label, secondsLeft)
				app.QueueUpdateDraw(func() {
					vGainersLosers.SetTitle(title)
					vGainersLosers.SetText(c)
				})
				phase = (phase + 1) % 4
			}
			refreshGL()
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				secondsLeft--
				if secondsLeft <= 0 {
					secondsLeft = gainersSecs
					refreshGL()
					continue
				}
				// Title reflects current content (phase was already advanced in refreshGL)
				displayPhase := (phase + 3) % 4
				label := "gainers"
				if displayPhase >= 2 {
					label = "losers"
				}
				app.QueueUpdateDraw(func() {
					vGainersLosers.SetTitle(fmt.Sprintf(" Crypto %s (%ds) ", label, secondsLeft))
				})
			}
		}()
		// Crypto News: one headline + article per page, 20s cycle; timer in title; pool refresh every 6s
		go func() {
			newsSecs := 20
			secondsLeft := newsSecs
			pageIndex := 0
			items, _ := fetchCryptoNewsItems(baseURL)
			tickCount := 0

			refreshContent := func() {
				ch := make(chan struct{ w, h int }, 1)
				app.QueueUpdateDraw(func() {
					_, _, w, h := vNews.GetRect()
					if w < 20 {
						w = 60
					}
					if h < 3 {
						h = 24
					}
					ch <- struct{ w, h int }{w: w, h: h - 2}
					close(ch)
				})
				size := <-ch
				n := len(items)
				if n > 0 {
					if pageIndex >= n {
						pageIndex = 0
					}
					c := renderCryptoNewsTwoItems(items, pageIndex, size.w, size.h)
					title := fmt.Sprintf(" Crypto News (%ds) ", secondsLeft)
					app.QueueUpdateDraw(func() {
						vNews.SetTitle(title)
						vNews.SetText(c)
					})
				} else {
					app.QueueUpdateDraw(func() {
						vNews.SetTitle(" Crypto News ")
						vNews.SetText("No headlines")
					})
				}
			}

			refreshContent()
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				tickCount++
				// Refresh pool every 6s
				if tickCount%6 == 0 {
					if newItems, err := fetchCryptoNewsItems(baseURL); err == nil && len(newItems) > 0 {
						items = newItems
						if pageIndex >= len(items) {
							pageIndex = 0
						}
					}
				}
				secondsLeft--
				if secondsLeft <= 0 {
					secondsLeft = newsSecs
					if len(items) > 0 {
						pageIndex = (pageIndex + 2) % len(items)
					}
					refreshContent()
					continue
				}
				title := fmt.Sprintf(" Crypto News (%ds) ", secondsLeft)
				app.QueueUpdateDraw(func() {
					vNews.SetTitle(title)
				})
			}
		}()
		// BTC ETF Tracker: show all ETFs at once, refresh every 6s, live countdown in title
		go func() {
			refreshSecs := 6
			secondsLeft := refreshSecs
			refreshBtc := func() {
				c := renderCryptoBtcEtfAll(baseURL)
				title := fmt.Sprintf(" BTC ETF Tracker (%ds) ", secondsLeft)
				app.QueueUpdateDraw(func() {
					vBtc.SetTitle(title)
					vBtc.SetText(c)
				})
			}
			refreshBtc()
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				secondsLeft--
				if secondsLeft <= 0 {
					secondsLeft = refreshSecs
					refreshBtc()
					continue
				}
				title := fmt.Sprintf(" BTC ETF Tracker (%ds) ", secondsLeft)
				app.QueueUpdateDraw(func() {
					vBtc.SetTitle(title)
				})
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
