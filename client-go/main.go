// Pi Terminal World Monitor – Go client using tview.
// Fetches panel data from the backend; displays a responsive grid of panels.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	defaultBackendURL = "http://localhost:8000"
	defaultCycleSecs  = 8
	requestTimeout    = 10 * time.Second
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
	Conditions string `json:"conditions"`
}

type weatherResp struct {
	Status    string       `json:"status"`
	Message   string       `json:"message"`
	Locations []weatherLoc `json:"locations"`
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
		return []string{"world-clock", "weather"}, nil
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

func renderWeather(baseURL string) string {
	var d weatherResp
	if err := fetchJSON(baseURL, "/panels/weather", &d); err != nil {
		return "No data"
	}
	if d.Status == "placeholder" && d.Message != "" {
		return d.Message
	}
	if len(d.Locations) == 0 {
		return "No locations yet."
	}
	var b strings.Builder
	for _, loc := range d.Locations {
		b.WriteString(fmt.Sprintf("  %s: %s° %s\n", loc.Name, loc.Temp, loc.Conditions))
	}
	return strings.TrimSuffix(b.String(), "\n")
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

	app := tview.NewApplication()

	// Build grid: 2 columns, 2 rows by default; fill with panels in order
	cols, rows := 2, 2
	n := cols * rows
	if n > len(panels) {
		// repeat panels to fill
		for len(panels) < n {
			panels = append(panels, panels...)
		}
		panels = panels[:n]
	} else {
		panels = panels[:n]
	}

	// Grid: cols x rows for panels, then one full-width row for footer
	grid := tview.NewGrid().
		SetColumns(-1, -1).
		SetRows(-1, -1, 1).
		SetBorders(false)

	// One TextView per cell; we'll update them on refresh
	textViews := make([]*tview.TextView, n)
	for i := 0; i < n; i++ {
		key := panels[i]
		title := panelTitle(key)
		content := panelContent(baseURL, key)

		tv := tview.NewTextView().
			SetDynamicColors(true).
			SetText(content)
		tv.SetBorder(true).SetTitle(" " + title + " ")

		textViews[i] = tv
		row, col := i/cols, i%cols
		grid.AddItem(tv, row, col, 1, 1, 0, 0, false)
	}

	// Footer row
	footer := tview.NewTextView().SetText(fmt.Sprintf(" Next in %ds · Q quit · Ctrl+C exit ", cycleSecs)).SetTextAlign(tview.AlignCenter)
	footer.SetBorder(false)
	grid.AddItem(footer, rows, 0, 1, cols, 0, 0, false)

	// Refresh all panels on a timer
	go func() {
		ticker := time.NewTicker(time.Duration(cycleSecs) * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			for i := 0; i < n; i++ {
				key := panels[i]
				content := panelContent(baseURL, key)
				idx := i
				app.QueueUpdate(func() {
					textViews[idx].SetText(content)
				})
			}
			app.QueueUpdate(func() {
				footer.SetText(fmt.Sprintf(" Next in %ds · Q quit · Ctrl+C exit ", cycleSecs))
			})
		}
	}()

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
