package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

func main() {
	url := "http://localhost:7077"
	username := "user"
	password := "user"

	fmt.Printf("🚀 Starting Load Generator against %s\n", url)
	fmt.Printf("👥 Workers: 20 | Iterations: 50\n\n")

	// 1. Get Token
	loginBody, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	resp, err := http.Post(url+"/login", "application/json", bytes.NewBuffer(loginBody))
	if err != nil {
		fmt.Printf("❌ Initial login failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ Initial login failed with status %d\n", resp.StatusCode)
		return
	}

	var loginResp map[string]string
	json.NewDecoder(resp.Body).Decode(&loginResp)
	token := loginResp["access_token"]

	const workerCount = 20
	const iterations = 50
	var wg sync.WaitGroup

	start := time.Now()
	successCount := 0
	var mu sync.Mutex

	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func(id int) {
			defer wg.Done()
			client := &http.Client{Timeout: 5 * time.Second}
			for j := 0; j < iterations; j++ {
				req, _ := http.NewRequest("GET", url+"/news?limit=5", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				
				res, err := client.Do(req)
				if err != nil {
					continue
				}
				io.Copy(io.Discard, res.Body)
				res.Body.Close()
				
				if res.StatusCode == http.StatusOK {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
				
				if (id == 0 && j%10 == 0) {
					fmt.Printf("   Worker 0: %d/%d completed...\n", j, iterations)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	total := workerCount * iterations
	
	fmt.Printf("\n🏁 Load Test Complete!\n")
	fmt.Printf("⏱️  Duration: %v\n", duration)
	fmt.Printf("✅ Success: %d/%d (%.1f%%)\n", successCount, total, float64(successCount)/float64(total)*100)
	fmt.Printf("⚡ Average: %.2f req/sec\n", float64(total)/duration.Seconds())
}
