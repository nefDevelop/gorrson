package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nef734/gorrson/internal/auth"
	"github.com/nef734/gorrson/internal/db"
)

// --- V5: autenticar no puede crear bases de datos -------------------------

// TestLoginNoCreaBasesDeDatos es la regresión de la creación de bases de datos no
// autenticada: /login es público y getStore() abría y migraba una base de datos
// nueva para CUALQUIER nombre de usuario, así que un POST bastaba para escribir
// ~50 KB en disco por intento y dejar un *sql.DB cacheado para siempre.
func TestLoginNoCreaBasesDeDatos(t *testing.T) {
	t.Setenv("GORRSON_TEST", "true")
	dataDir := t.TempDir()
	srv := NewServer(dataDir, true)

	for _, usuario := range []string{"no_existe", "inventado1", "inventado2"} {
		body := `{"username":"` + usuario + `","password":"lo_que_sea"}`
		rec := httptestJSON(body)
		req := newJSONRequest(rec, "/login", body)
		srv.router.ServeHTTP(rec, req)

		if rec.Code == 0 {
			t.Fatalf("no se llegó a escribir respuesta")
		}
		if _, err := os.Stat(filepath.Join(dataDir, usuario+".db")); err == nil {
			t.Fatalf("el login creó la base de datos %s.db; debe dar 401 sin crear nada", usuario)
		}
	}

	// Y el mensaje no debe revelar si el usuario existe.
	entries, _ := os.ReadDir(dataDir)
	if len(entries) != 0 {
		t.Errorf("se escribieron %d ficheros autenticando contra usuarios inexistentes", len(entries))
	}
}

// --- V4: el login está limitado -------------------------------------------

// TestLoginTieneLimiteDeIntentos comprueba que /login deja de ser un amplificador
// de memoria: cada intento cuesta un argon2id de ~67 MB.
func TestLoginTieneLimiteDeIntentos(t *testing.T) {
	t.Setenv("GORRSON_TEST", "true")
	dataDir := t.TempDir()
	srv := NewServer(dataDir, true)

	store, err := db.NewStore(dataDir, "victima")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	hash, _ := auth.HashPassword("la_correcta")
	if err := store.UpsertUser(t.Context(), "victima", hash, ""); err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	store.Close()

	body := `{"username":"victima","password":"incorrecta"}`
	limite := srv.loginLimiter.max

	vistos := map[int]int{}
	for i := 0; i < limite+5; i++ {
		rec := httptestJSON(body)
		srv.router.ServeHTTP(rec, newJSONRequest(rec, "/login", body))
		vistos[rec.Code]++
	}

	if vistos[429] == 0 {
		t.Errorf("nunca se devolvió 429 tras %d intentos fallidos (por IP: %v). "+
			"El límite no estáfrenando los intentos de login.", limite+5, vistos)
	}
	if vistos[401] > limite {
		t.Errorf("se ejecutaron %d verificaciones de contraseña; el máximo es %d por ventana",
			vistos[401], limite)
	}
}

// TestLoginCorrectoLiberaElContador: un usuario legítimo que se equivoca unas
// veces y luego entra bien no debe quedarse bloqueado.
func TestLoginCorrectoLiberaElContador(t *testing.T) {
	t.Setenv("GORRSON_TEST", "true")
	dataDir := t.TempDir()
	srv := NewServer(dataDir, true)

	store, _ := db.NewStore(dataDir, "usuario_ok")
	hash, _ := auth.HashPassword("correcta")
	_ = store.UpsertUser(t.Context(), "usuario_ok", hash, "")
	store.Close()

	for i := 0; i < srv.loginLimiter.max-1; i++ {
		body := `{"username":"usuario_ok","password":"mal"}`
		rec := httptestJSON(body)
		srv.router.ServeHTTP(rec, newJSONRequest(rec, "/login", body))
		if rec.Code != 401 {
			t.Fatalf("intento %d: se esperaba 401, llegó %d", i, rec.Code)
		}
	}

	body := `{"username":"usuario_ok","password":"correcta"}`
	rec := httptestJSON(body)
	srv.router.ServeHTTP(rec, newJSONRequest(rec, "/login", body))
	if rec.Code == 429 {
		t.Error("un login correcto quedó bloqueado por los intentos fallidos anteriores")
	}
}

// --- V1: el proxy de imágenes no sirve HTML --------------------------------

// TestProxyDeImagenesNoSirveHTML es la regresión de servir HTML del atacante
// desde el origen de la app. Con ServeContent y el nombre "image" (sin
// extensión) se hacía DetectContentType sobre los primeros bytes y salía
// text/html; con 'unsafe-inline' en la CSP global el <script> se ejecutaba.
func TestProxyDeImagenesNoSirveHTML(t *testing.T) {
	t.Setenv("GORRSON_TEST", "true")
	dataDir := t.TempDir()
	srv := NewServer(dataDir, true)

	username := "proxy_victima"
	store, _ := db.NewStore(dataDir, username)
	hash, _ := auth.HashPassword("x")
	_ = store.UpsertUser(t.Context(), username, hash, "")
	store.Close()
	token, _ := auth.GenerateToken(username)

	html := "<!DOCTYPE html><html><body><script>alert(document.domain)</script>robo</body></html>"
	origen := nuevoServidorDeImagenes(t, "text/html", []byte(html))

	rec := httptestJSON("")
	req := newProxyRequest(origen, token)
	srv.router.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); strings.HasPrefix(ct, "text/html") {
		t.Errorf("el proxy devolvió %q; un atacante podría servir HTML desde el origen de la app", ct)
	}
	if rec.Code == 200 && strings.Contains(rec.Body.String(), "<script>") {
		t.Error("el proxy devolvió el <script> del atacante en el cuerpo")
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("falta X-Content-Type-Options: nosniff, que es lo que impide que el navegador interprete el cuerpo")
	}
	if rec.Code == 200 {
		t.Error("un recurso declarado text/html debería rechazarse con 415, no servirse")
	}
}

// TestProxyDeImagenesAceptaSVGYAVIF: http.DetectContentType NO reconoce SVG ni
// AVIF, que son miniaturas perfectamente normales en un lector de RSS. Si el
// proxy solo mirara los bytes, las rechazaría.
func TestProxyDeImagenesAceptaSVG(t *testing.T) {
	t.Setenv("GORRSON_TEST", "true")
	dataDir := t.TempDir()
	srv := NewServer(dataDir, true)

	username := "proxy_svg"
	store, _ := db.NewStore(dataDir, username)
	hash, _ := auth.HashPassword("x")
	_ = store.UpsertUser(t.Context(), username, hash, "")
	store.Close()
	token, _ := auth.GenerateToken(username)

	origen := nuevoServidorDeImagenes(t, "image/svg+xml",
		[]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="1" height="1"></svg>`))

	rec := httptestJSON("")
	srv.router.ServeHTTP(rec, newProxyRequest(origen, token))

	if rec.Code != 200 {
		t.Errorf("un SVG legítimo fue rechazado con %d; DetectContentType no lo reconoce "+
			"y hay que fiarse de la cabecera del origen", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Errorf("Content-Type = %q; want image/svg+xml", ct)
	}
}

// --- V11: importar un backup no puede dejar escrituras en un inode muerto ---

// TestEvictStoreDejaDeVerLosDatosViejos: tras sustituir el fichero por fuera, el
// *sql.DB cacheado sigue abierto sobre el inode antiguo (sin nombre pero válido),
// así que las lecturas devolvían los datos viejos y las escrituras se perdían.
func TestEvictStoreDejaDeVerLosDatosViejos(t *testing.T) {
	t.Setenv("GORRSON_TEST", "true")
	dataDir := t.TempDir()
	srv := NewServer(dataDir, true)
	username := "backup_user"

	store, err := srv.getStore(username)
	if err != nil {
		t.Fatalf("getStore: %v", err)
	}
	if err := store.UpsertUser(t.Context(), username, "hash-antiguo", ""); err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}

	// Simula lo que hace handleImportBackup: sustituir el fichero por fuera.
	srv.evictStore(username)

	nuevo, err := srv.getStore(username)
	if err != nil {
		t.Fatalf("getStore tras evict: %v", err)
	}
	defer nuevo.Close()
	got, err := nuevo.GetUserHash(t.Context(), username)
	if err != nil {
		t.Fatalf("GetUserHash: %v", err)
	}
	if got != "hash-antiguo" {
		t.Errorf("hash leído = %q; want hash-antiguo (el store nuevo debe ver el fichero actual)", got)
	}
}

// --- helpers ---------------------------------------------------------------

func httptestJSON(body string) *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

func newJSONRequest(rec *httptest.ResponseRecorder, path, body string) *http.Request {
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func newProxyRequest(origen, token string) *http.Request {
	req := httptest.NewRequest("GET", "/proxy?url="+origen, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func nuevoServidorDeImagenes(t *testing.T, contentType string, body []byte) string {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(body)
	}))
	t.Cleanup(ts.Close)
	return ts.URL
}
