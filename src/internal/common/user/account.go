package user

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/elecbug/linuxus/src/internal/common/ruleset"
	"golang.org/x/crypto/bcrypt"
)

// Legacy bcrypt entries remain readable. Updated entries also carry account
// state and a random generation so unlocking never revives old session cookies.
type Account struct {
	Maintenance bool   `json:"maintenance,omitempty"`
	Hash        string `json:"hash"`
	Locked      bool   `json:"locked,omitempty"`
	Generation  string `json:"generation,omitempty"`
	Template    string `json:"template,omitempty"`
	Class       string `json:"class,omitempty"`
}

const accountPrefix = "!linuxus1!"

func ParseAccount(value string) (Account, error) {
	if !strings.HasPrefix(value, accountPrefix) {
		return Account{Hash: value}, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, accountPrefix))
	var account Account
	if err == nil {
		err = json.Unmarshal(data, &account)
	}
	if err != nil || account.Hash == "" || account.Generation == "" {
		return Account{}, fmt.Errorf("invalid account record")
	}
	return account, nil
}

func IsLocked(value string) bool {
	account, err := ParseAccount(value)
	return err != nil || account.Locked || account.Maintenance
}

func CheckPassword(value, password string) error {
	account, err := ParseAccount(value)
	if err != nil {
		return err
	}
	err = bcrypt.CompareHashAndPassword([]byte(account.Hash), []byte(password))
	if account.Locked || account.Maintenance {
		return fmt.Errorf("account locked")
	}
	return err
}

// UpdateAccount preserves the auth inode and unrelated lines under the same
// cross-process lock as signup. Every successful change revokes old cookies.
func UpdateAccount(path, id, operation, value string) error {
	if !ruleset.AllowedUserID(id) {
		return fmt.Errorf("invalid user ID")
	}
	if operation == "password" {
		if value == "" {
			return fmt.Errorf("password must not be empty")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(value), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		value = string(hash)
	}
	file, err := openLocked(path, os.O_RDWR, syscall.LOCK_EX)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	users, err := readUsers(bytes.NewReader(data))
	if err != nil {
		return err
	}
	record, ok := users[id]
	if !ok {
		return fmt.Errorf("user %q does not exist", id)
	}
	account, err := ParseAccount(record)
	if err != nil {
		return err
	}
	if account.Maintenance && operation != "maintenance-end" && operation != "maintenance-failed" {
		return fmt.Errorf("account is under disk maintenance; wait or use recover-user after the operation exits")
	}
	switch operation {
	case "maintenance":
		account.Maintenance = true
	case "maintenance-end":
		account.Maintenance = false
	case "maintenance-failed":
		account.Maintenance = false
		account.Locked = true
	case "lock":
		account.Locked = true
	case "unlock":
		account.Locked = false
	case "password":
		account.Hash = value
	case "disconnect": // Rotate the cookie generation only.
	case "template":
		account.Template = value
	case "class":
		account.Class = value
		account.Template = ""
	default:
		return fmt.Errorf("unsupported account operation")
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	account.Generation = hex.EncodeToString(nonce[:])
	encoded, err := json.Marshal(account)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		key, _, ok := strings.Cut(strings.TrimSpace(line), ":")
		if ok && strings.TrimSpace(key) == id {
			lines[i] = id + ":" + accountPrefix + base64.RawURLEncoding.EncodeToString(encoded)
		}
	}
	return replaceAuthData(file, data, []byte(strings.Join(lines, "\n")))
}
