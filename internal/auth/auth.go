package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

// Argon2id parameters (OWASP recommended)
const (
	Memory      = 64 * 1024 // 64 MB
	Iterations  = 3
	Parallelism = 2
	SaltLength  = 16
	KeyLength   = 32
)

// EnvSecretVar es la variable de entorno con el secreto de firma de los JWT.
const EnvSecretVar = "GORRSON_JWT_SECRET"

// minSecretLength: los tokens viven 24 h, así que la clave tiene que ser
// criptográficamente fuerte. 32 bytes es el tamaño de HMAC-SHA256.
const minSecretLength = 32

// forbiddenSecrets son valores PÚBLICOS: están en el código fuente o en el
// Dockerfile, así que un despliegue que arranque con ellos tiene la autenticación
// rota — cualquiera puede firmar un JWT como cualquier usuario y acceder a
// /backup/export. Se rechazan en vez de aceptarlos en silencio.
var forbiddenSecrets = []string{
	"gorrson-vibrant-rss-secret", // literal que hubo en este repo
	"change-me-in-production",    // Dockerfile
	"cambia-este-secreto-largo-y-aleatorio-de-mas-de-32-caracteres", // placeholder de .env.example
	"changeme",
	"secret",
	"password",
	"admin",
	"jwt-secret",
}

var (
	secretMu     sync.Mutex
	secret       []byte
	secretRandom bool // generado para esta ejecución: las sesiones no sobreviven a un reinicio
)

// InitSecret valida el secreto JWT y debe llamarse antes de servir tráfico.
//
// Tres casos, a propósito:
//
//   - Secreto configurado: se usa. Si es uno de los valores públicos conocidos se
//     devuelve error, para que el despliegue falle de forma ruidosa en vez de
//     arrancar con la autenticación rota.
//   - Sin secreto configurado: se genera uno aleatorio de 32 bytes y se avisa por
//     log. No es un fallo de arranque porque obligaría a declarar el secreto
//     también en desarrollo, pero desde luego no se vuelve a usar un valor
//     público: la diferencia es que los tokens dejan de sobrevivir a un reinicio.
//   - Secreto demasiado corto: error, por la misma razón que los públicos.
func InitSecret() error {
	secretMu.Lock()
	defer secretMu.Unlock()

	env := strings.TrimSpace(os.Getenv(EnvSecretVar))

	if env == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return fmt.Errorf("no se pudo generar el secreto JWT: %w", err)
		}
		secret = b
		secretRandom = true
		slog.Warn("Seguridad: "+EnvSecretVar+" no está definido, se genera un secreto "+
			"aleatorio para esta ejecución. Las sesiones se invalidarán al reiniciar y "+
			"los tokens firmados con el valor por defecto dejarán de ser válidos. "+
			"Define la variable antes de desplegar.",
			"longitud_minima", minSecretLength)
		return nil
	}

	for _, bad := range forbiddenSecrets {
		if strings.EqualFold(env, bad) {
			return fmt.Errorf("%s tiene el valor público %q: con él, cualquiera puede firmar "+
				"un JWT como cualquier usuario. Genera uno con `openssl rand -base64 32`",
				EnvSecretVar, bad)
		}
	}
	if len(env) < minSecretLength {
		return fmt.Errorf("%s tiene %d caracteres; se exigen al menos %d. "+
			"Genera uno con `openssl rand -base64 32`",
			EnvSecretVar, len(env), minSecretLength)
	}

	secret = []byte(env)
	secretRandom = false
	return nil
}

// SecretIsRandom indica si el secreto en uso se generó al vuelo porque no había
// ninguno configurado. Si es true, reiniciar invalida todas las sesiones.
func SecretIsRandom() bool {
	secretMu.Lock()
	defer secretMu.Unlock()
	return secretRandom
}

// GetSecret devuelve el secreto de firma.
//
// Si InitSecret no se ha llamado (tests, o un embedding que se olvide de hacerlo),
// se inicializa de forma perezosa para no dejar la aplicación sin funcionar; el
// servidor ya la llama en el arranque y aborta si falla.
func GetSecret() []byte {
	secretMu.Lock()
	haySecret := len(secret) > 0
	secretMu.Unlock()

	if !haySecret {
		if err := InitSecret(); err != nil {
			// Invocado sin SecretInit() y con un secreto inválido: no hay forma
			// segura de seguir. Se usa un secreto aleatorio para no firmar con un
			// valor conocido, y el error se propaga en ValidateToken.
			b := make([]byte, 32)
			_, _ = rand.Read(b)
			secretMu.Lock()
			secret = b
			secretRandom = true
			secretMu.Unlock()
		}
	}

	secretMu.Lock()
	defer secretMu.Unlock()
	return secret
}

// HashPassword generates an argon2id hash in PHC format.
func HashPassword(password string) (string, error) {
	salt := make([]byte, SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, Iterations, Memory, Parallelism, KeyLength)

	// Format: $argon2id$v=19$m=65536,t=3,p=2$salt$hash
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encodedHash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, Memory, Iterations, Parallelism, b64Salt, b64Hash)

	return encodedHash, nil
}

// CheckPasswordHash verifies an argon2id hash.
func CheckPasswordHash(password, encodedHash string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false
	}

	var memory uint32
	var iterations uint32
	var parallelism uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	comparisonHash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(hash)))

	return subtle.ConstantTimeCompare(hash, comparisonHash) == 1
}

// GenerateToken creates a short-lived JWT (Access Token) for the given username.
func GenerateToken(username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // 24 hours
	})

	tokenString, err := token.SignedString(GetSecret())
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenString, nil
}

// GenerateRefreshToken creates a long-lived secure random string.
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ValidateToken verifies a JWT and returns the associated username.
func ValidateToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return GetSecret(), nil
	})

	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		username, ok := claims["username"].(string)
		if !ok {
			return "", errors.New("token payload missing username")
		}
		return username, nil
	}

	return "", errors.New("token invalid or expired")
}
