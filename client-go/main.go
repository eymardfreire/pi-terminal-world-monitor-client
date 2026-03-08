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
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
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
	b.WriteString(fmt.Sprintf(" %s%s[-]", statusTag, d.StatusLabel))
	if d.MarketCapB != nil && d.VolumeB != nil {
		b.WriteString(fmt.Sprintf("  MCap: $%.1fB | Vol: $%.1fB", *d.MarketCapB, *d.VolumeB))
	}
	b.WriteString("\n\n [::b]PEG HEALTH[-]\n")
	for _, c := range d.Coins {
		pegTag := "[green]"
		if c.PegStatus != "ON PEG" {
			pegTag = "[red]"
		}
		b.WriteString(fmt.Sprintf("  [::b]%s[-] %s  $%.4f  %s%s[-] %.2f%%\n", c.Symbol, c.Name, c.Price, pegTag, c.PegStatus, c.DeviationPct))
	}
	b.WriteString("\n [::b]SUPPLY & VOLUME[-]\n")
	for _, c := range d.Coins {
		mcapStr := "—"
		if c.MarketCapB != nil {
			if *c.MarketCapB >= 1 {
				mcapStr = fmt.Sprintf("$%.1fB", *c.MarketCapB)
			} else {
				mcapStr = fmt.Sprintf("$%.0fM", *c.MarketCapB*1000)
			}
		}
		volStr := formatStableVol(c.VolumeB)
		chgStr := "—"
		if c.PriceChange24hPct != nil {
			chgStr = fmt.Sprintf("%+.2f%%", *c.PriceChange24hPct)
		}
		chgTag := "[green]"
		if c.PriceChange24hPct != nil && *c.PriceChange24hPct < 0 {
			chgTag = "[red]"
		}
		b.WriteString(fmt.Sprintf("  [::b]%s[-]  %s  %s  %s%s[-]\n", c.Symbol, mcapStr, volStr, chgTag, chgStr))
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func renderCryptoGainersLosers(baseURL string, showGainers bool) string {
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
	var b strings.Builder
	b.WriteString(fmt.Sprintf(" [::b]Crypto %s[-]\n\n", title))
	for _, e := range list {
		priceStr := fmt.Sprintf("%.4f", e.Price)
		if e.Price >= 1000 {
			priceStr = fmt.Sprintf("%.2f", e.Price)
		} else if e.Price >= 1 {
			priceStr = fmt.Sprintf("%.2f", e.Price)
		}
		b.WriteString(fmt.Sprintf("  %s%-8s[-] $%s\n", tag, e.Symbol, priceStr))
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

// renderCryptoNewsOneArticle appends one article (headline wrapped, no truncation + blurb) into b, up to maxLines. Returns lines used.
func renderCryptoNewsOneArticle(b *strings.Builder, it *cryptoNewsItem, width, maxLines int) int {
	if maxLines <= 0 {
		return 0
	}
	indent := "  "
	// Headline: wrap to width (no ellipsis), allow multiple lines
	titleLines := wrapLines(strings.TrimSpace(it.Title), width-2)
	maxTitleLines := 2
	if maxTitleLines > maxLines-2 {
		maxTitleLines = maxLines - 2 // need at least 1 blank + 1 blurb
	}
	linesUsed := 0
	for i, w := range titleLines {
		if i >= maxTitleLines {
			break
		}
		b.WriteString(indent)
		b.WriteString(w)
		b.WriteString("\n")
		linesUsed++
	}
	b.WriteString("\n")
	linesUsed++
	if linesUsed >= maxLines {
		return linesUsed
	}
	blurb := it.Description
	if blurb == "" {
		blurb = "—"
	}
	wrapped := wrapLines(blurb, width-2)
	remaining := maxLines - linesUsed
	for i, w := range wrapped {
		if i >= remaining {
			break
		}
		b.WriteString(indent)
		b.WriteString(w)
		b.WriteString("\n")
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

	// One TextView per cell (or a Flex for crypto with 5 sub-panels: Top | Stable+GainersLosers; News | BTC ETF)
	textViews := make([]*tview.TextView, n)
	var cryptoSubpanelViews [5]*tview.TextView
	for i := 0; i < n; i++ {
		key := panels[i]
		row, col := i/cols, i%cols
		if key == "crypto" {
			// Crypto: Top | (Stablecoins | Gainers/Losers); Crypto News | BTC ETF
			tvTop := tview.NewTextView().SetDynamicColors(true).SetText(renderCryptoTop(baseURL, 1))
			tvTop.SetBorder(true).SetTitle(" Top cryptos by mcap (8s) ")
			tvStable := tview.NewTextView().SetDynamicColors(true).SetText(renderCryptoStablecoins(baseURL))
			tvStable.SetBorder(true).SetTitle(" Stablecoins ")
			tvGainersLosers := tview.NewTextView().SetDynamicColors(true).SetText(renderCryptoGainersLosers(baseURL, true))
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

	// Crypto panel: 5 sub-panels. Top | (Stablecoins | Gainers/Losers); News | BTC ETF.
	if cryptoPanelIndex >= 0 && cryptoSubpanelViews[0] != nil {
		vTop := cryptoSubpanelViews[0]
		vStable := cryptoSubpanelViews[1]
		vGainersLosers := cryptoSubpanelViews[2]
		vNews := cryptoSubpanelViews[3]
		vBtc := cryptoSubpanelViews[4]
		// Top Cryptos: static title "Top cryptos by mcap (8s)", 8s per page; lines per page from panel height (resolution-aware)
		go func() {
			perPage := 11
			rangeStarts := []int{1, 12, 23}
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
					numPages := (33 + perPage - 1) / perPage
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
		// Stablecoins: refresh every 6s with timer in title
		go func() {
			stableSecs := 6
			secondsLeft := stableSecs
			refreshStable := func() {
				c := renderCryptoStablecoins(baseURL)
				title := fmt.Sprintf(" Stablecoins (%ds) ", secondsLeft)
				app.QueueUpdateDraw(func() {
					vStable.SetTitle(title)
					vStable.SetText(c)
				})
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
		// Gainers/Losers: cycle every 10s between gainers (green) and losers (red)
		go func() {
			gainersSecs := 10
			secondsLeft := gainersSecs
			showGainers := true
			refreshGL := func() {
				c := renderCryptoGainersLosers(baseURL, showGainers)
				label := "gainers"
				if !showGainers {
					label = "losers"
				}
				title := fmt.Sprintf(" Crypto %s (%ds) ", label, secondsLeft)
				app.QueueUpdateDraw(func() {
					vGainersLosers.SetTitle(title)
					vGainersLosers.SetText(c)
				})
			}
			refreshGL()
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				secondsLeft--
				if secondsLeft <= 0 {
					secondsLeft = gainersSecs
					showGainers = !showGainers
					refreshGL()
					continue
				}
				label := "gainers"
				if !showGainers {
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
