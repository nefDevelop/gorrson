package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"
)

type Result struct {
	Duration time.Duration
	Err      error
	Type     string
}

func main() {
	url := "http://localhost:7077"
	username := "user"
	password := "user"

	fmt.Printf("🚀 Gorrson Benchmark Tool\n")
	fmt.Printf("Target: %s\n", url)

	token, err := login(url, username, password)
	if err != nil {
		fmt.Printf("❌ Login failed: %v\n", err)
		return
	}

	const workers = 15
	const iterations = 100
	results := make(chan Result, workers*iterations)

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			client := &http.Client{Timeout: 10 * time.Second}
			for j := 0; j < iterations; j++ {
				// Mix of operations
				var opType string
				var opErr error
				opStart := time.Now()

				switch (workerID + j) % 4 {
				case 0:
					opType = "LIST_NEWS"
					opErr = doRequest(client, "GET", url+"/news?limit=20&unread=true", token)
				case 1:
					opType = "SEARCH"
					opErr = doRequest(client, "GET", url+"/news?search=tech", token)
				case 2:
					opType = "STATS"
					opErr = doRequest(client, "GET", url+"/stats", token)
				case 3:
					opType = "LIST_FEEDS"
					opErr = doRequest(client, "GET", url+"/feeds", token)
				}

				results <- Result{
					Duration: time.Since(opStart),
					Err:      opErr,
					Type:     opType,
				}
			}
		}(i)
	}

	wg.Wait()
	close(results)
	totalDuration := time.Since(start)

	report(results, totalDuration)
}

func login(baseURL, user, pass string) (string, error) {
	loginBody, _ := json.Marshal(map[string]string{
		"username": user,
		"password": pass,
	})
	resp, err := http.Post(baseURL+"/login", "application/json", bytes.NewBuffer(loginBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	var loginResp map[string]string
	json.NewDecoder(resp.Body).Decode(&loginResp)
	return loginResp["access_token"], nil
}

func doRequest(client *http.Client, method, targetURL, token string) error {
	req, _ := http.NewRequest(method, targetURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

func report(results chan Result, totalDuration time.Duration) {
	var allDurations []time.Duration
	byType := make(map[string][]time.Duration)
	errors := 0

	for r := range results {
		if r.Err != nil {
			errors++
			continue
		}
		allDurations = append(allDurations, r.Duration)
		byType[r.Type] = append(byType[r.Type], r.Duration)
	}

	fmt.Printf("\n🏁 Benchmark Results\n")
	fmt.Printf("Total Duration: %v\n", totalDuration)
	fmt.Printf("Total Requests: %d\n", len(allDurations)+errors)
	fmt.Printf("Success Rate:   %.2f%%\n", float64(len(allDurations))/float64(len(allDurations)+errors)*100)
	fmt.Printf("Throughput:     %.2f req/sec\n", float64(len(allDurations))/totalDuration.Seconds())

	printStats("Overall", allDurations)
	
	keys := make([]string, 0, len(byType))
	for k := range byType {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		printStats(k, byType[k])
	}
}

func printStats(name string, durations []time.Duration) {
	if len(durations) == 0 {
		return
	}
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	p50 := durations[len(durations)/2]
	p95 := durations[len(durations)*95/100]
	p99 := durations[len(durations)*99/100]
	avg := 0 * time.Nanosecond
	for _, d := range durations {
		avg += d
	}
	avg /= time.Duration(len(durations))

	fmt.Printf("\n📊 [%s]\n", name)
	fmt.Printf("  Avg: %v\n", avg)
	fmt.Printf("  P50: %v\n", p50)
	fmt.Printf("  P95: %v\n", p95)
	fmt.Printf("  P99: %v\n", p99)
}
