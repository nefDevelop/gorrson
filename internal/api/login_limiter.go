package api

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// Limitador de intentos de autenticación.
//
// El login ejecuta un argon2id por intento: con los parámetros del proyecto
// (auth.go: 64 MiB, t=3, p=2) son ~62 ms y ~67 MB de RAM por verificación. Sin
// límite, 100 logins concurrentes desde una sola IP son 6,7 GB y ~1,6 s de CPU,
// y no hace falta autenticarse para provocarlo: /login es público.
//
// Se cuentan por IP y por usuario, porque un atacante distribuido que no
// repita IP no tocaría nunca el límite de IP.
type loginLimiter struct {
	mu      sync.Mutex
	porIP   map[string]*intento
	porUser map[string]*intento
	max     int
	ventana time.Duration
	nowFn   func() time.Time
}

type intento struct {
	conteo int
	inicio time.Time
}

func newLoginLimiter(max int, ventana time.Duration) *loginLimiter {
	return &loginLimiter{
		porIP:   make(map[string]*intento),
		porUser: make(map[string]*intento),
		max:     max,
		ventana: ventana,
		nowFn:   time.Now,
	}
}

// permitir registra un intento y dice si se puede seguir.
//
// Se cuenta el intento ANTES de comprobar la contraseña a propósito: si solo se
// contaran los fallidos, un atacante que acierte la contraseña pero luegoceive un
// 401 por otra causa (o que solo intente logins correctos con otra contraseña
// para medir) no contaría.
func (l *loginLimiter) permitir(clave string, m map[string]*intento) bool {
	ahora := l.nowFn()

	l.mu.Lock()
	defer l.mu.Unlock()

	e, ok := m[clave]
	if !ok || ahora.Sub(e.inicio) > l.ventana {
		m[clave] = &intento{conteo: 1, inicio: ahora}
		l.purgar(m, ahora)
		return true
	}

	e.conteo++
	return e.conteo <= l.max
}

// liberar descuenta el intento cuando la autenticación sí fue correcta: un usuario
// legítimo que se equivoca tres veces y luego entra bien no debe quedarse con la
// cuenta bloqueada hasta que expire la ventana.
func (l *loginLimiter) liberar(clave string, m map[string]*intento) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(m, clave)
}

// purgar limpia las claves cuya ventana ya venció, para que el mapa no crezca sin
// límite con IPs nuevas. Se ejecuta solo al crear una entrada nueva.
func (l *loginLimiter) purgar(m map[string]*intento, ahora time.Time) {
	if len(m) < 4096 {
		return
	}
	for k, e := range m {
		if ahora.Sub(e.inicio) > l.ventana {
			delete(m, k)
		}
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// permitirLogin combina los dos límites y devuelve el motivo si se rechaza.
func (s *Server) permitirLogin(r *http.Request, username string) (bool, string) {
	if s.loginLimiter.permitir(clientIP(r), s.loginLimiter.porIP) {
		if s.loginLimiter.permitir(username, s.loginLimiter.porUser) {
			return true, ""
		}
		return false, "demasiados intentos para ese usuario; inténtalo más tarde"
	}
	return false, "demasiados intentos desde esa dirección; inténtalo más tarde"
}

func (s *Server) loginLiberado(r *http.Request, username string) {
	s.loginLimiter.liberar(clientIP(r), s.loginLimiter.porIP)
	s.loginLimiter.liberar(username, s.loginLimiter.porUser)
}
