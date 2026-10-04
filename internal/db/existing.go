package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nef734/gorrson/internal/sanitizer"
)

// ErrUserNotFound se devuelve cuando no existe la base de datos de un usuario.
//
// Existe para poder responder "usuario no registrado" sin crear el fichero: antes,
// autenticar contra un usuario inexistente abría y migraba una base de datos
// nueva, así que un POST /login sin autenticar bastaba para escribir en disco (~50
// KB por usuario) y dejar un *sql.DB vivo en el mapa de stores.
var ErrUserNotFound = errors.New("user database not found")

// dbFilePath es la ruta de la base de datos de un usuario, sin abrirla.
func dbFilePath(dataDir, username string) (string, error) {
	if !sanitizer.IsValidUsername(username) {
		return "", fmt.Errorf("invalid username: %s", username)
	}
	return filepath.Join(dataDir, username+".db"), nil
}

// StoreExists dice si el usuario ya tiene base de datos, sin crearla ni migrarla.
//
// Es la comprobación que deben hacer los caminos de autenticación antes de abrir
// nada: permitir que un atacante cree bases de datos a voluntad es escritura en
// disco no autenticada, y además cada store creada se cachea para siempre.
func StoreExists(dataDir, username string) (bool, error) {
	path, err := dbFilePath(dataDir, username)
	if err != nil {
		return false, err
	}
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return !fi.IsDir(), nil
}

// OpenExistingStore abre la base de datos de un usuario SIN crearla y SIN migrarla.
//
// Se usa en los caminos donde crear la base de datos sería un efecto secundario
// no deseado (autenticación). Si el usuario no existe devuelve ErrUserNotFound.
//
// Migrar en cadarequest era además un coste real: 26 sentencias DDL por login y por
// petición con X-API-Key.
func OpenExistingStore(dataDir, username string) (*Store, error) {
	path, err := dbFilePath(dataDir, username)
	if err != nil {
		return nil, err
	}

	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if fi.IsDir() {
		return nil, ErrUserNotFound
	}

	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(5)

	return &Store{db: database, username: username}, nil
}

// StoreExistsIn indica si una tabla existe en la base de datos de un usuario, sin
// abrir el Store completo.
//
// handleFeverAPI necesita saber si un usuario tiene ya la tabla api_keys (y por
// tanto clave de Fever) antes de decidir si merece la pena verificar su
// contraseña. Sin esto, cada petición a /api/fever recorría TODOS los .db y
// ejecutaba un argon2 por usuario — 67 MB de RAM cada uno.
func StoreExistsIn(dataDir, username, table string) (bool, error) {
	store, err := OpenExistingStore(dataDir, username)
	if err != nil {
		return false, err
	}
	defer store.Close()

	var name string
	err = store.db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name = ?`, table).Scan(&name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
