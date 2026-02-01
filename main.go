package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	AppVersion  = "v1.0"
	ColorHeader = "\033[96m"
	ColorBlue   = "\033[94m"
	ColorGreen  = "\033[92m" 
	ColorRed    = "\033[91m" 
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
)

type ConfigEntry struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Token string `json:"token"`
}

type Site struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Threat struct {
	ID string `json:"id"`
}

type Alert struct {
	AlertInfo struct {
		AlertID string `json:"alertId"`
	} `json:"alertInfo"`
}

type SiteResponse struct {
	Data struct {
		Sites []Site `json:"sites"`
	} `json:"data"`
}

type ThreatResponse struct {
	Data []Threat `json:"data"`
}

type AlertResponse struct {
	Data []Alert `json:"data"`
}

type GenericResponse struct {
	Data struct {
		Affected int `json:"affected"`
	} `json:"data"`
}

type ScanResult struct {
	SiteName string
	Count    int
	Items    []string // IDs
}

// --- GLOBAL CONFIG ---
var client = &http.Client{Timeout: 15 * time.Second}

func main() {
	// 1. Parse Flags
	consoleFlag := flag.String("c", "", "Console Name")
	defaultFlag := flag.Bool("d", false, "Select Default Site")
	threatsFlag := flag.Bool("t", false, "Threats Mode")
	alertsFlag := flag.Bool("a", false, "Alerts Mode")
	listConsolesFlag := flag.Bool("list-consoles", false, "List consoles for autocomplete")
	versionFlag := flag.Bool("version", false, "Show version")
	flag.Parse()

	// 2. Handle Version
	if *versionFlag {
		fmt.Printf("S1Resolver %s\n", AppVersion)
		os.Exit(0)
	}

	// 3. Load Config
	config := loadConfig()

	// --- AUTOCOMPLETE HOOK ---
	if *listConsolesFlag {
		for _, c := range config {
			fmt.Println(c.Name)
		}
		os.Exit(0)
	}

	if len(config) == 0 {
		fmt.Printf("%sError: config.json not found or empty.%s\n", ColorRed, ColorReset)
		return
	}

	var activeConsole ConfigEntry
	var selectedSite Site

	// --- STEP 1: SELECT CONSOLE ---
	if *consoleFlag != "" {
		// Filter matches
		var matches []ConfigEntry
		search := strings.ToLower(*consoleFlag)
		for _, c := range config {
			if strings.Contains(strings.ToLower(c.Name), search) {
				matches = append(matches, c)
			}
		}

		if len(matches) == 0 {
			fmt.Printf("%sNo console found matching '%s'%s\n", ColorRed, *consoleFlag, ColorReset)
			return
		} else if len(matches) == 1 {
			activeConsole = matches[0]
			fmt.Printf("Console Selected: %s%s%s\n", ColorBold, activeConsole.Name, ColorReset)
		} else {
			fmt.Printf("\n%sMultiple consoles match '%s':%s\n", ColorHeader, *consoleFlag, ColorReset)
			for i, m := range matches {
				fmt.Printf("%s%d.%s %s\n", ColorBlue, i+1, ColorReset, m.Name)
			}
			idx := promptInt("\n"+ColorBold+"Select match (number):"+ColorReset+" ", 1, len(matches))
			activeConsole = matches[idx-1]
		}
	} else {
		// Manual Selection
		fmt.Printf("\n%s%s--- Select Console ---%s\n", ColorBold, ColorHeader, ColorReset)
		for i, c := range config {
			fmt.Printf("%s%d.%s %s\n", ColorBlue, i+1, ColorReset, c.Name)
		}
		idx := promptInt("\n"+ColorBold+"Select console (number):"+ColorReset+" ", 1, len(config))
		activeConsole = config[idx-1]
	}

	// --- STEP 2: SELECT SITE ---
	sites := getSites(activeConsole.URL, activeConsole.Token)
	if len(sites) == 0 {
		fmt.Printf("%sNo sites found or connection failed.%s\n", ColorRed, ColorReset)
		return
	}

	// Sort sites alphabetically
	sort.Slice(sites, func(i, j int) bool {
		return strings.ToLower(sites[i].Name) < strings.ToLower(sites[j].Name)
	})

	if *defaultFlag {
		for _, s := range sites {
			if strings.Contains(strings.ToLower(s.Name), "default") {
				selectedSite = s
				fmt.Printf("Auto-selected Site: %s%s%s\n", ColorBold, s.Name, ColorReset)
				goto STEP3
			}
		}
		fmt.Printf("%sNo 'Default' site found.%s\n", ColorRed, ColorReset)
		return
	} else {
		// Loops back to select site if alert or threat was preselected using flags without a site selecion
		for {
			fmt.Printf("\n%s%s--- Available Sites ---%s\n", ColorBold, ColorHeader, ColorReset)
			fmt.Printf("%s0.%s %sNot Sure (Scan All Sites)%s\n", ColorBlue, ColorReset, ColorBold, ColorReset)
			for i, s := range sites {
				fmt.Printf("%s%d.%s %s\n", ColorBlue, i+1, ColorReset, s.Name)
			}

			inputStr := promptString("\n" + ColorBold + "Select Site:" + ColorReset + " ")

			// Option to scan all sites simultaneously
			if inputStr == "0" {
				mode := ""
				if *threatsFlag {
					mode = "1"
				} else if *alertsFlag {
					mode = "2"
				} else {
					mode = promptString(fmt.Sprintf("1. %sThreats%s\n2. %sAlerts%s\nChoice: ", ColorRed, ColorReset, ColorRed, ColorReset))
				}

				fmt.Printf("Scanning %d sites...\n", len(sites))

				// Worker Pool Config
				maxConcurrency := 5
				sem := make(chan struct{}, maxConcurrency)
				results := make(chan ScanResult, len(sites))
				var wg sync.WaitGroup

				for _, s := range sites {
					wg.Add(1)
					go func(site Site) {
						defer wg.Done()
						sem <- struct{}{}        // Acquire token
						defer func() { <-sem }() // Release token

						var foundIDs []string
						if mode == "1" {
							items := getThreats(activeConsole.URL, activeConsole.Token, site.ID, true)
							for _, it := range items {
								foundIDs = append(foundIDs, it.ID)
							}
						} else {
							items := getAlerts(activeConsole.URL, activeConsole.Token, site.ID, true)
							for _, it := range items {
								foundIDs = append(foundIDs, it.AlertInfo.AlertID)
							}
						}

						if len(foundIDs) > 0 {
							results <- ScanResult{SiteName: site.Name, Count: len(foundIDs), Items: foundIDs}
						}
					}(s)
				}

				// Closer routine
				go func() {
					wg.Wait()
					close(results)
				}()

				// Process Results
				foundAny := false
				for res := range results {
					foundAny = true
					lbl := "Threats"
					if mode == "2" {
						lbl = "Alerts"
					}
					fmt.Printf("Found %d %s%s%s in %s%s%s\n", res.Count, ColorRed, lbl, ColorReset, ColorBold, res.SiteName, ColorReset)
				}

				if !foundAny {
					fmt.Printf("%sScan Complete. No items found.%s\n", ColorGreen, ColorReset)
				} else {
					fmt.Printf("\n%sScan Complete.%s\n", ColorGreen, ColorReset)
				}
				
				fmt.Println("\nPress Enter to return to site list...")
				fmt.Scanln()
				continue // <--- LOOPS BACK TO MENU
			}

			// Regular Selection
			if idx, err := strconv.Atoi(inputStr); err == nil && idx > 0 && idx <= len(sites) {
				selectedSite = sites[idx-1]
				break // <--- BREAKS LOOP TO CONTINUE
			} else {
				fmt.Printf("%sInvalid selection.%s\n", ColorRed, ColorReset)
				// Loops back to try again
			}
		}
	}

STEP3:
	fmt.Printf("\nSelected: %s%s%s\n", ColorBold, selectedSite.Name, ColorReset)

	// --- STEP 3: ACTION ---
	mode := "0"
	if *threatsFlag {
		mode = "1"
	} else if *alertsFlag {
		mode = "2"
	} else {
		fmt.Printf("\n%s%s--- Action ---%s\n", ColorBold, ColorHeader, ColorReset)
		fmt.Printf("%s1.%s Threats\n", ColorBlue, ColorReset)
		fmt.Printf("%s2.%s Alerts\n", ColorBlue, ColorReset)
		mode = promptString("Choice: ")
	}

	if mode == "1" {
		threats := getThreats(activeConsole.URL, activeConsole.Token, selectedSite.ID, false)
		if len(threats) == 0 {
			fmt.Printf("%sNo undefined threats found.%s\n", ColorBlue, ColorReset)
			return
		}
		fmt.Printf("\nFound %s%d%s undefined threats.\n", ColorBold, len(threats), ColorReset)
		if promptYesNo(fmt.Sprintf("Mark ALL as %sFalse Positive & Resolved%s? (y/n): ", ColorGreen, ColorReset)) {
			note := promptString("Resolution Note: ")
			var ids []string
			for _, t := range threats {
				ids = append(ids, t.ID)
			}
			resolveThreats(activeConsole.URL, activeConsole.Token, ids, note)
		}
	} else if mode == "2" {
		alerts := getAlerts(activeConsole.URL, activeConsole.Token, selectedSite.ID, false)
		if len(alerts) == 0 {
			fmt.Printf("%sNo undefined alerts found.%s\n", ColorBlue, ColorReset)
			return
		}
		fmt.Printf("\nFound %s%d%s undefined alerts.\n", ColorBold, len(alerts), ColorReset)
		if promptYesNo(fmt.Sprintf("Mark ALL as %sFalse Positive & Resolved%s? (y/n): ", ColorGreen, ColorReset)) {
			var ids []string
			for _, a := range alerts {
				ids = append(ids, a.AlertInfo.AlertID)
			}
			resolveAlerts(activeConsole.URL, activeConsole.Token, ids)
		}
	} else {
		fmt.Println("Invalid choice.")
	}
}

// --- API FUNCTIONS ---

func getSites(baseURL, token string) []Site {
	url := baseURL + "/web/api/v2.1/sites?limit=1000&state=active"
	body, err := makeRequest("GET", url, token, nil)
	if err != nil {
		return nil
	}

	var resp SiteResponse
	json.Unmarshal(body, &resp)
	return resp.Data.Sites
}

func getThreats(baseURL, token, siteID string, silent bool) []Threat {
	if !silent {
		fmt.Printf("Fetching active %s%sthreats%s...\n", ColorBold, ColorRed, ColorReset)
	}
	url := baseURL + "/web/api/v2.1/threats?limit=1000&analystVerdicts=undefined&incidentStatuses=unresolved&siteIds=" + siteID
	body, err := makeRequest("GET", url, token, nil)
	if err != nil {
		return nil
	}
	var resp ThreatResponse
	json.Unmarshal(body, &resp)
	return resp.Data
}

func getAlerts(baseURL, token, siteID string, silent bool) []Alert {
	if !silent {
		fmt.Printf("Fetching active %s%salerts%s...\n", ColorBold, ColorRed, ColorReset)
	}
	url := baseURL + "/web/api/v2.1/cloud-detection/alerts?limit=1000&analystVerdict=UNDEFINED&incidentStatus=UNRESOLVED&siteIds=" + siteID
	body, err := makeRequest("GET", url, token, nil)
	if err != nil {
		return nil
	}
	var resp AlertResponse
	json.Unmarshal(body, &resp)
	return resp.Data
}

func resolveThreats(baseURL, token string, ids []string, note string) {
	fmt.Println(" - Adding resolution note...")
	payloadNote := map[string]interface{}{
		"filter": map[string][]string{"ids": ids},
		"data":   map[string]string{"text": note},
	}
	makeRequest("POST", baseURL+"/web/api/v2.1/threats/notes", token, payloadNote)

	fmt.Println(" - Updating status to FP and Resolved...")
	payloadResolve := map[string]interface{}{
		"filter": map[string][]string{"ids": ids},
		"data":   map[string]string{"incidentStatus": "resolved", "analystVerdict": "false_positive"},
	}
	resp, _ := makeRequest("POST", baseURL+"/web/api/v2.1/threats/incident", token, payloadResolve)

	var gResp GenericResponse
	json.Unmarshal(resp, &gResp)
	count := len(ids)
	if gResp.Data.Affected > 0 {
		count = gResp.Data.Affected
	}

	fmt.Printf(" - %sSuccess!%s %d threats marked as FP & Resolved.\n", ColorGreen, ColorReset, count)
}

func resolveAlerts(baseURL, token string, ids []string) {
	fmt.Println(" - Setting verdict to False Positive...")
	payloadVerdict := map[string]interface{}{
		"filter": map[string][]string{"ids": ids},
		"data":   map[string]string{"analystVerdict": "FALSE_POSITIVE"},
	}
	makeRequest("POST", baseURL+"/web/api/v2.1/cloud-detection/alerts/analyst-verdict", token, payloadVerdict)

	fmt.Println(" - Setting status to Resolved...")
	payloadStatus := map[string]interface{}{
		"filter": map[string][]string{"ids": ids},
		"data":   map[string]string{"incidentStatus": "RESOLVED"},
	}
	makeRequest("POST", baseURL+"/web/api/v2.1/cloud-detection/alerts/incident", token, payloadStatus)

	fmt.Printf(" - %sSuccess!%s Alerts marked as FP & Resolved.\n", ColorGreen, ColorReset)
}

// --- HELPER FUNCTIONS ---

func makeRequest(method, url, token string, data interface{}) ([]byte, error) {
	var body io.Reader
	if data != nil {
		jsonBytes, _ := json.Marshal(data)
		body = bytes.NewBuffer(jsonBytes)
	}

	req, _ := http.NewRequest(method, url, body)
	req.Header.Set("Authorization", "ApiToken "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("API Error: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func loadConfig() []ConfigEntry {
	var config []ConfigEntry
	var file []byte
	var err error

	// Try executable directory
	ex, _ := os.Executable()
	exePath := filepath.Join(filepath.Dir(ex), "config.json")
	file, err = os.ReadFile(exePath)

	// ORRR try current working directory
	if err != nil {
		wd, _ := os.Getwd()
		wdPath := filepath.Join(wd, "config.json")
		file, err = os.ReadFile(wdPath)
	}

	if err != nil {
		return nil
	}

	json.Unmarshal(file, &config)
	return config
}

func promptString(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func promptInt(prompt string, min, max int) int {
	for {
		s := promptString(prompt)
		i, err := strconv.Atoi(s)
		if err == nil && i >= min && i <= max {
			return i
		}
		fmt.Printf("%sInvalid number. Try again.%s\n", ColorRed, ColorReset)
	}
}

func promptYesNo(prompt string) bool {
	s := strings.ToLower(promptString(prompt))
	return s == "y" || s == "yes"
}
