package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/nef734/gorrson/internal/auth"
	"github.com/nef734/gorrson/internal/db"
	"github.com/nef734/gorrson/internal/models"
)

type stats struct {
	mu       sync.Mutex
	latencies []time.Duration
}

func (s *stats) record(d time.Duration) {
	s.mu.Lock()
	s.latencies = append(s.latencies, d)
	s.mu.Unlock()
}

func (s *stats) report(name string, totalOps int, totalDuration time.Duration) {
	if len(s.latencies) == 0 {
		return
	}
	var sum time.Duration
	min := s.latencies[0]
	max := s.latencies[0]
	for _, l := range s.latencies {
		sum += l
		if l < min {
			min = l
		}
		if l > max {
			max = l
		}
	}
	avg := sum / time.Duration(len(s.latencies))
	rps := float64(totalOps) / totalDuration.Seconds()

	fmt.Printf("\n📊 --- %s ---\n", name)
	fmt.Printf("⏱️  Total Duration: %v\n", totalDuration)
	fmt.Printf("🚀 Requests/sec:   %.2f\n", rps)
	fmt.Printf("📟 Avg Latency:    %v\n", avg)
	fmt.Printf("⏬ Min Latency:    %v\n", min)
	fmt.Printf("⏫ Max Latency:    %v\n", max)
	fmt.Printf("------------------------------\n")
}

func TestStress_ConcurrentLoginAndList(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	dataDir := "./test_stress_data"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir)
	username := "stress_tester"
	password := "stress_pass"

	store, _ := db.NewStore(dataDir, username)
	hash, _ := auth.HashPassword(password)
	store.UpsertUser(username, hash, "")
	store.Close()

	// Pre-generate token to avoid redundant hashing overhead in listing test
	token, _ := auth.GenerateToken(username)

	const workerCount = 50
	const iterations = 100
	var wg sync.WaitGroup

	// --- Phase 1: Login ---
	loginStats := &stats{}
	start := time.Now()
	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				loginBody, _ := json.Marshal(map[string]string{
					"username": username,
					"password": password,
				})
				req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(loginBody))
				rr := httptest.NewRecorder()
				
				opStart := time.Now()
				srv.router.ServeHTTP(rr, req)
				loginStats.record(time.Since(opStart))
			}
		}()
	}
	wg.Wait()
	loginStats.report("Login Stress", workerCount*iterations, time.Since(start))

	// --- Phase 2: Listing ---
	listStats := &stats{}
	startListing := time.Now()
	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				req := httptest.NewRequest("GET", "/news?limit=20", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				rr := httptest.NewRecorder()
				
				opStart := time.Now()
				srv.router.ServeHTTP(rr, req)
				listStats.record(time.Since(opStart))
			}
		}()
	}
	wg.Wait()
	listStats.report("Listing Stress", workerCount*iterations, time.Since(startListing))
}

func TestStress_DiskSearch(t *testing.T) {
	dataDir := "./test_stress_search"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir)
	username := "search_tester"
	store, _ := db.NewStore(dataDir, username)
	
	// Seed 1000 entries
	for i := 0; i < 1000; i++ {
		store.AddEntry(&models.Entry{
			ID:          fmt.Sprintf("e%d", i),
			FeedID:      "f1",
			Title:       "Title " + fmt.Sprint(i),
			Content:     "Some content with keyword",
			URL:         "url" + fmt.Sprint(i),
			PublishedAt: time.Now(),
		})
	}
	store.Close()
	token, _ := auth.GenerateToken(username)

	const workerCount = 20
	const iterations = 50
	var wg sync.WaitGroup
	
	searchStats := &stats{}
	start := time.Now()

	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				req := httptest.NewRequest("GET", "/news?search=keyword", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				rr := httptest.NewRecorder()
				
				opStart := time.Now()
				srv.router.ServeHTTP(rr, req)
				searchStats.record(time.Since(opStart))
			}
		}()
	}
	wg.Wait()
	searchStats.report("Disk Search Stress", workerCount*iterations, time.Since(start))
}
