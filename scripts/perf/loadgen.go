//go:build ignore

// loadgen — generador de carga y medidor de RSS para gorrson.
//
// A diferencia de regio, gorrson SÍ tiene endpoints propios, así que aquí no hace
// falta upstream dummy: se golpea el servidor real con la base ya sembrada.
//
// RSS se lee de /proc/<pid>/status (VmRSS actual, VmHWM pico). Sin dependencias.
//
// Uso:
//
//	go run scripts/perf/loadgen.go [flags]
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	_ "modernc.org/sqlite"
)

// ---------------------------------------------------------------------------
// Flags
// ---------------------------------------------------------------------------

var (
	binary    = flag.String("binary", "./rss-cli", "Binario de gorrson a medir")
	dataDir   = flag.String("data-dir", "./data/perf", "Directorio de bases de datos por usuario")
	username  = flag.String("user", "user", "Usuario sembrado")
	password  = flag.String("pass", "user", "Contraseña del usuario")
	port      = flag.Int("port", 17077, "Puerto del servidor")
	profiles  = flag.String("profiles", "health,news,feeds,search,search1,write", "Perfiles: health,news,feeds,search,search1,write")
	levels    = flag.String("levels", "10,50,100,200", "Niveles de concurrencia")
	duration  = flag.Duration("duration", 10*time.Second, "Duración medida por fase")
	warmup    = flag.Duration("warmup", 3*time.Second, "Calentamiento por fase (no se contabiliza)")
	outJSON   = flag.String("out", "", "Fichero JSON de resultados")
	tokenFile = flag.String("token-file", "", "Fichero temporal donde volcar el token (diagnóstico)")
)

// ---------------------------------------------------------------------------
// Modelos de resultados
// ---------------------------------------------------------------------------

type sample struct {
	ts  time.Time
	rss int64
	hwm int64
}

type phaseResult struct {
	Profile       string           `json:"profile"`
	Level         int              `json:"concurrency"`
	Duration      string           `json:"duration"`
	Requests      int64            `json:"requests"`
	Errors        int64            `json:"errors"`
	Codes         map[string]int64 `json:"status_codes"`
	RPS           float64          `json:"rps"`
	LatP50Ms      float64          `json:"lat_p50_ms"`
	LatP95Ms      float64          `json:"lat_p95_ms"`
	LatP99Ms      float64          `json:"lat_p99_ms"`
	LatMaxMs      float64          `json:"lat_max_ms"`
	BytesOut      int64            `json:"bytes_out"`
	RSSStartBytes int64            `json:"rss_start_bytes"`
	RSSEndBytes   int64            `json:"rss_end_bytes"`
	RSSPeakBytes  int64            `json:"rss_peak_bytes"`
	RSSDeltaBytes int64            `json:"rss_delta_bytes"`
}

type report struct {
	Timestamp  string         `json:"timestamp"`
	GoVersion  string         `json:"go_version"`
	GOOS       string         `json:"goos"`
	NumCPU     int            `json:"num_cpu"`
	GOMAXPROCS int            `json:"gomaxprocs"`
	DataDir    string         `json:"data_dir"`
	DBPath     string         `json:"db_path"`
	DBBytes    int64          `json:"db_bytes"`
	DBBytesEnd int64          `json:"db_bytes_after_load"`
	Entries    int64          `json:"entries"`
	ServerPID  int            `json:"server_pid"`
	IdleRSS    int64          `json:"idle_rss_bytes"`
	IdleHWM    int64          `json:"idle_hwm_bytes"`
	SeedNote   string         `json:"seed_note"`
	Profiles   map[string]any `json:"profiles_measured"`
	Phases     []phaseResult  `json:"phases"`
}

type phase struct {
	profile string
	level   int
}

// ---------------------------------------------------------------------------

var authToken string

func main() {
	flag.Parse()

	dbFile := filepath.Join(*dataDir, *username+".db")
	if _, err := os.Stat(dbFile); err != nil {
		fatal("no existe %s (ejecuta `make perf-seed` antes): %v", dbFile, err)
	}
	dbBefore := fileSize(dbFile)

	srv, err := startServer()
	if err != nil {
		fatal("arrancando %s: %v", *binary, err)
	}
	pid := srv.Process.Pid
	defer stopServer(srv)

	base := fmt.Sprintf("http://127.0.0.1:%d", *port)
	if err := waitHealthy(base, 25*time.Second); err != nil {
		fatal("gorrson no respondió /health: %v", err)
	}
	fmt.Printf("  gorrson         pid=%d  :%d\n", pid, *port)

	token, err := login(base)
	if err != nil {
		fatal("login: %v", err)
	}
	authToken = token
	if *tokenFile != "" {
		_ = os.WriteFile(*tokenFile, []byte(token), 0o600)
	}
	fmt.Printf("  login           ok (Bearer %s...)\n", trunc(token, 12))

	entries := countEntries(dbFile)
	entriesTotal = int(entries)
	if entriesTotal < 1 {
		entriesTotal = 1
	}
	fmt.Printf("  dataset         %.1f MiB, %d entradas\n", mib(dbBefore), entries)

	if err := loadEntryIDs(dbFile, 5000); err != nil {
		fatal("leyendo ids de entrada para el perfil write: %v", err)
	}
	fmt.Printf("  ids para write  %d\n", len(entryIDs))

	// RSS en reposo: referencia obligatoria. Sin ella "RSS 90 MiB" no dice nada,
	// porque incluye lo que el proceso ya tenía antes de la carga.
	time.Sleep(2 * time.Second)
	idleRSS, idleHWM := readRSS(pid)
	fmt.Printf("  RSS en reposo   %.1f MiB (VmHWM %.1f MiB)\n", mib(idleRSS), mib(idleHWM))

	phases := buildPhases()
	results := make([]phaseResult, 0, len(phases))
	for _, ph := range phases {
		res := runPhase(base, pid, ph)
		results = append(results, res)
		printPhase(res)
		time.Sleep(400 * time.Millisecond)
	}

	rep := report{
		Timestamp:  time.Now().Format(time.RFC3339),
		GoVersion:  runtime.Version(),
		GOOS:       runtime.GOOS,
		NumCPU:     runtime.NumCPU(),
		GOMAXPROCS: runtime.GOMAXPROCS(0),
		DataDir:    *dataDir,
		DBPath:     dbFile,
		DBBytes:    dbBefore,
		DBBytesEnd: fileSize(dbFile),
		Entries:    entries,
		ServerPID:  pid,
		IdleRSS:    idleRSS,
		IdleHWM:    idleHWM,
		SeedNote: "La entidad escalable es `entries` (con su índice FTS5, vía trigger " +
			"entries_ai): es lo que miden /news y /news?search=. Los feeds se quedan en " +
			"42 porque son los que tiene un usuario real y se usan como filtro " +
			"(category, feed_id), no como volumen.",
		Profiles: map[string]any{
			"health": "GET /health. Sin autenticación y sin tocar la DB: acota el coste " +
				"del router, de los middlewares (Compress(5)) y del JSON, por diferencia.",
			"news": "GET /news?limit=50 con Bearer. La lectura más caliente: JOIN con " +
				"feeds y ORDER BY published_at sobre 50 entradas serializadas.",
			"search": "GET /news?search=<término común>&limit=50 con Bearer. Peor caso de " +
				"ORDER BY rank: el término está en el 100 % de las entradas, así que FTS5 " +
				"puntúa y ordena todas las coincidencias.",
			"search1": "GET /news?search=docNNNNNNNN con Bearer, el token único que el seed " +
				"escribe en cada entrada. Mejor caso: una sola coincidencia. Comparado con " +
				"`search` acota cuánto del coste es ORDER BY rank y cuánto es el recorrido.",
			"write": "PUT /news/{id}/read. Escritura sobre SQLite con WAL y " +
				"SetMaxOpenConns(1): mide la serialización de las escrituras.",
		},
		Phases: results,
	}

	out := *outJSON
	if out == "" {
		if err := os.MkdirAll("perf", 0o755); err != nil {
			fatal("creando perf/: %v", err)
		}
		out = filepath.Join("perf", "load-"+time.Now().Format("20060102-150405")+".json")
	}
	blob, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		fatal("serializando: %v", err)
	}
	if err := os.WriteFile(out, blob, 0o644); err != nil {
		fatal("escribiendo %s: %v", out, err)
	}
	fmt.Printf("\n  DB tras carga   %.1f MiB (%+.1f MiB)\n",
		mib(rep.DBBytesEnd), mib(rep.DBBytesEnd-rep.DBBytes))
	fmt.Printf("  resultados      %s\n", out)
}

// ---------------------------------------------------------------------------
// Servidor bajo prueba
// ---------------------------------------------------------------------------

func startServer() (*exec.Cmd, error) {
	cmd := exec.Command(*binary, "serve",
		"--port", strconv.Itoa(*port),
		"--data-dir", *dataDir,
		// --silent quita el middleware.Logger: sin él cada petición escribe en
		// stderr y el generador ahogaría el disco. El logging se mide aparte, en
		// el perfil health con el servidor real.
		"--silent",
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

func stopServer(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	done := make(chan struct{})
	go func() { cmd.Wait(); close(done) }() //nolint:errcheck
	select {
	case <-done:
	case <-time.After(12 * time.Second):
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

func waitHealthy(base string, timeout time.Duration) error {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		resp, err := client.Get(base + "/health")
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			last = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			last = err
		}
		time.Sleep(200 * time.Millisecond)
	}
	if last == nil {
		last = fmt.Errorf("timeout")
	}
	return last
}

// login hace un único POST /login y devuelve el access_token. El token se reutiliza
// en todas las peticiones: si el login estuviera dentro del bucle, bcrypt dominaría
// el RPS y estaríamos midiendo el KDF en vez del servidor (mismo criterio que en
// regio).
func login(base string) (string, error) {
	body, _ := json.Marshal(map[string]string{"username": *username, "password": *password})
	resp, err := http.Post(base+"/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	blob, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, trunc(string(blob), 200))
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(blob, &out); err != nil {
		return "", err
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("respuesta sin access_token")
	}
	return out.AccessToken, nil
}

// ---------------------------------------------------------------------------
// RSS
// ---------------------------------------------------------------------------

func sampleRSS(pid int, stop <-chan struct{}, out chan<- sample) {
	t := time.NewTicker(50 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			rss, hwm := readRSS(pid)
			if rss == 0 {
				continue
			}
			select {
			case out <- sample{ts: time.Now(), rss: rss, hwm: hwm}:
			default:
			}
		}
	}
}

func readRSS(pid int) (rss, hwm int64) {
	blob, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/status")
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.Split(string(blob), "\n") {
		switch {
		case strings.HasPrefix(line, "VmRSS:"):
			rss = parseKB(line)
		case strings.HasPrefix(line, "VmHWM:"):
			hwm = parseKB(line)
		}
	}
	return rss, hwm
}

func parseKB(line string) int64 {
	f := strings.Fields(line)
	if len(f) < 2 {
		return 0
	}
	v, err := strconv.ParseInt(f[1], 10, 64)
	if err != nil {
		return 0
	}
	return v * 1024
}

func peakOf(samples []sample) int64 {
	var peak int64
	for _, s := range samples {
		if s.rss > peak {
			peak = s.rss
		}
		if s.hwm > peak {
			peak = s.hwm
		}
	}
	return peak
}

// ---------------------------------------------------------------------------
// Fases
// ---------------------------------------------------------------------------

func buildPhases() []phase {
	var profs []string
	for _, p := range strings.Split(*profiles, ",") {
		if p = strings.TrimSpace(p); p != "" {
			profs = append(profs, p)
		}
	}
	var lv []int
	for _, l := range strings.Split(*levels, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(l)); err == nil && n > 0 {
			lv = append(lv, n)
		}
	}
	out := make([]phase, 0, len(profs)*len(lv))
	for _, p := range profs {
		for _, l := range lv {
			out = append(out, phase{profile: p, level: l})
		}
	}
	return out
}

// searchTerms escalonados: si todos los workers buscaran lo mismo, FTS5 devolvería
// siempre el mismo conjunto con la página más caliente cacheada. Cada worker rota
// por un término distinto para repartir la carga.
var searchTerms = []string{
	"gorrson", "benchmark", "rendimiento", "concurrencia", "SQLite",
	"latencia", "throughput", "observabilidad", "kernel", "seguridad",
	"algoritmos", "contenedores", "profiling", "métricas", "compiladores",
}

// El perfil `search` usa términos que están en TODAS las entradas: es el peor caso
// de ORDER BY rank, porque FTS5 tiene que puntuar y ordenar el 100 % de las
// coincidencias. El perfil `search1` usa el token único (docNNNNNNNN) que el seed
// escribe en cada entrada, así que devuelve una sola fila: el mejor caso. Medir los
// dos extremos es lo que hace que el número signifique algo.
func uniqueTokenFor(worker, iter int) string {
	return fmt.Sprintf("doc%08d", (worker*100003+iter*7919)%entriesTotal)
}

// entriesTotal lo fija main() con el conteo real del dataset.
var entriesTotal = 1

type stats struct {
	reqs  atomic.Int64
	errs  atomic.Int64
	bytes atomic.Int64
	codes sync.Map // status code (string) -> *atomic.Int64
	mu    sync.Mutex
	lat   []time.Duration
}

func (s *stats) noteCode(code int) {
	key := strconv.Itoa(code)
	v, _ := s.codes.LoadOrStore(key, new(atomic.Int64))
	v.(*atomic.Int64).Add(1)
}

func (s *stats) codeBreakdown() map[string]int64 {
	out := map[string]int64{}
	s.codes.Range(func(k, v any) bool {
		key := k.(string)
		if strings.HasPrefix(key, "2") {
			return true
		}
		out[key] = v.(*atomic.Int64).Load()
		return true
	})
	return out
}

func runPhase(base string, pid int, ph phase) phaseResult {
	if *warmup > 0 {
		s := &stats{}
		runWorkers(base, ph, s, *warmup)
		time.Sleep(250 * time.Millisecond)
	}

	c := &stats{}
	res := phaseResult{Profile: ph.profile, Level: ph.level}
	res.RSSStartBytes, _ = readRSS(pid)

	stop := make(chan struct{})
	samplesCh := make(chan sample, 8192)
	go sampleRSS(pid, stop, samplesCh)

	t0 := time.Now()
	runWorkers(base, ph, c, *duration)
	elapsed := time.Since(t0)

	res.RSSEndBytes, _ = readRSS(pid)
	time.Sleep(120 * time.Millisecond)
	close(stop)
	res.RSSPeakBytes = peakOf(drain(samplesCh))

	res.Requests = c.reqs.Load()
	res.Errors = c.errs.Load()
	res.BytesOut = c.bytes.Load()
	res.Codes = c.codeBreakdown()
	res.RPS = float64(res.Requests) / elapsed.Seconds()
	res.Duration = elapsed.Round(time.Millisecond).String()
	res.RSSDeltaBytes = res.RSSEndBytes - res.RSSStartBytes

	c.mu.Lock()
	sort.Slice(c.lat, func(i, j int) bool { return c.lat[i] < c.lat[j] })
	res.LatP50Ms = pctMs(c.lat, 0.50)
	res.LatP95Ms = pctMs(c.lat, 0.95)
	res.LatP99Ms = pctMs(c.lat, 0.99)
	if n := len(c.lat); n > 0 {
		res.LatMaxMs = float64(c.lat[n-1].Microseconds()) / 1000
	}
	c.mu.Unlock()

	return res
}

func runWorkers(base string, ph phase, c *stats, d time.Duration) {
	var wg sync.WaitGroup
	deadline := time.Now().Add(d)
	for w := 0; w < ph.level; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			client := newClient()
			i := 0
			for time.Now().Before(deadline) {
				i++
				status, n, dur, err := fire(client, base, ph.profile, worker, i)
				if err != nil {
					c.errs.Add(1)
					continue
				}
				c.reqs.Add(1)
				c.bytes.Add(n)
				c.noteCode(status)
				if status >= 400 {
					c.errs.Add(1)
				}
				c.mu.Lock()
				c.lat = append(c.lat, dur)
				c.mu.Unlock()
			}
		}(w)
	}
	wg.Wait()
}

// entryIDs lo rellena loadEntryIDs antes de las fases: el perfil `write` necesita
// ids reales de entrada.
var entryIDs []string

func fire(client *http.Client, base, profile string, worker, iter int) (int, int64, time.Duration, error) {
	var (
		req *http.Request
		err error
	)

	switch profile {
	case "health":
		req, err = http.NewRequest("GET", base+"/health", nil)
	case "search":
		term := searchTerms[(worker*3+iter)%len(searchTerms)]
		req, err = http.NewRequest("GET",
			base+"/news?search="+term+"&limit=50", nil)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+authToken)
		}
	case "search1":
		req, err = http.NewRequest("GET",
			base+"/news?search="+uniqueTokenFor(worker, iter)+"&limit=50", nil)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+authToken)
		}
	case "feeds":
		// Lista los feeds con su contador de no leídos. El contador es un subplan
		// correlacionado (uno por feed), así que sin un índice que lo cubra
		// recorre la tabla de entries entera 42 veces por petición.
		req, err = http.NewRequest("GET", base+"/feeds", nil)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+authToken)
		}
	case "write":
		if len(entryIDs) == 0 {
			return 0, 0, 0, fmt.Errorf("sin ids de entrada para el perfil write")
		}
		id := entryIDs[(worker*13+iter)%len(entryIDs)]
		req, err = http.NewRequest("PUT", base+"/news/"+id+"/read", nil)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+authToken)
		}
	default: // "news"
		// unread + sortPublished es lo que hace el frontend al abrir la app.
		req, err = http.NewRequest("GET",
			base+"/news?limit=50&unread=true&sort=published", nil)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+authToken)
		}
	}
	if err != nil {
		return 0, 0, 0, err
	}
	if profile != "health" {
		req.Header.Set("User-Agent", "gorrson-loadgen/1.0")
	}

	t0 := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, time.Since(t0), err
	}
	// Con Compress(5) activo el cuerpo llega comprimido: se cuentan los bytes de
	// cable, no los de payload.
	n, _ := io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return resp.StatusCode, n, time.Since(t0), nil
}

func newClient() *http.Client {
	return &http.Client{
		Timeout: 20 * time.Second,
		// Sin seguir redirecciones y sin cookies: el token va en la cabecera y el
		// login no forma parte del bucle de carga.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			MaxIdleConns:        8192,
			MaxIdleConnsPerHost: 8192,
			MaxConnsPerHost:     8192,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}

func pctMs(sorted []time.Duration, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(p*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return float64(sorted[idx].Microseconds()) / 1000
}

func drain(ch chan sample) []sample {
	var out []sample
	for {
		select {
		case s := <-ch:
			out = append(out, s)
		default:
			return out
		}
	}
}

// ---------------------------------------------------------------------------
// Salida
// ---------------------------------------------------------------------------

func printPhase(r phaseResult) {
	fmt.Printf("\n  [%s] concurrencia=%-4d ventana=%s\n", r.Profile, r.Level, r.Duration)
	fmt.Printf("    req/s %-9.0f  p50 %-8.2fms  p95 %-8.2fms  p99 %-8.2fms  max %-8.2fms\n",
		r.RPS, r.LatP50Ms, r.LatP95Ms, r.LatP99Ms, r.LatMaxMs)
	fmt.Printf("    req %-10d err %-8d  salida %.1f MiB\n",
		r.Requests, r.Errors, mib(r.BytesOut))
	if len(r.Codes) > 0 {
		fmt.Printf("    códigos       %s\n", formatCodes(r.Codes))
	}
	fmt.Printf("    RSS inicio %.1f MiB → fin %.1f MiB (pico %.1f MiB, delta %+.1f MiB)\n",
		mib(r.RSSStartBytes), mib(r.RSSEndBytes), mib(r.RSSPeakBytes), mib(r.RSSDeltaBytes))
}

func formatCodes(codes map[string]int64) string {
	type kv struct {
		code string
		n    int64
	}
	list := make([]kv, 0, len(codes))
	for c, n := range codes {
		list = append(list, kv{c, n})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].n != list[j].n {
			return list[i].n > list[j].n
		}
		return list[i].code < list[j].code
	})
	parts := make([]string, 0, len(list))
	for _, e := range list {
		parts = append(parts, fmt.Sprintf("%s×%d", e.code, e.n))
	}
	return strings.Join(parts, " ")
}

// ---------------------------------------------------------------------------
// Utilidades
// ---------------------------------------------------------------------------

// openReadOnly abre la DB sembrada en solo lectura para leer metadatos (conteo de
// entradas e ids). El servidor tiene su propia conexión con WAL, y SQLite admite
// varios lectores simultáneos, así que no interfiere.
func openReadOnly(dbFile string) (*sql.DB, error) {
	return sql.Open("sqlite", "file:"+dbFile+"?mode=ro&_pragma=busy_timeout(5000)")
}

func countEntries(dbFile string) int64 {
	DB, err := openReadOnly(dbFile)
	if err != nil {
		return 0
	}
	defer DB.Close()
	var n int64
	if err := DB.QueryRow("SELECT COUNT(*) FROM entries").Scan(&n); err != nil {
		return 0
	}
	return n
}

// loadEntryIDs carga ids reales de entrada: el perfil `write` hace
// PUT /news/{id}/read y un id inventado devolvería 404, midiendo un error en vez
// de una escritura.
func loadEntryIDs(dbFile string, n int) error {
	DB, err := openReadOnly(dbFile)
	if err != nil {
		return err
	}
	defer DB.Close()

	// LIMIT para no arrastrar un millón de ids a memoria: los trabajadores eligen
	// al azar entre este conjunto, que basta para repartir escrituras.
	rows, err := DB.Query("SELECT id FROM entries ORDER BY published_at DESC LIMIT ?", n)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		entryIDs = append(entryIDs, id)
	}
	return rows.Err()
}

func fileSize(path string) int64 {
	fi, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return fi.Size()
}

func mib(b int64) float64 { return float64(b) / (1024 * 1024) }

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "loadgen: "+format+"\n", args...)
	os.Exit(1)
}
