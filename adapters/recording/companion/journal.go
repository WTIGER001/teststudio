package companion

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/recording"
)

type State struct {
	ID             string                       `json:"id"`
	Server         string                       `json:"server"`
	Instance       string                       `json:"instance"`
	DeviceToken    string                       `json:"deviceToken"`
	PairingToken   string                       `json:"pairingToken,omitempty"`
	Packet         *recording.CompanionPacket   `json:"packet,omitempty"`
	StartAttempted bool                         `json:"startAttempted"`
	Sequence       uint64                       `json:"sequence"`
	Last           recording.CompanionSnapshot  `json:"last"`
	Pending        *recording.CompanionSnapshot `json:"pending,omitempty"`
	Final          *recording.CompanionSnapshot `json:"final,omitempty"`
}

func (s State) String() string   { return "desktop recording: " + s.ID }
func (s State) GoString() string { return s.String() }

func randomToken() (string, error) {
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", errors.New("desktop credential generation failed")
	}
	return base64.RawURLEncoding.EncodeToString(token[:]), nil
}

func NewState(server, connection string) (State, error) {
	origin, err := Origin(server)
	if err != nil {
		return State{}, err
	}
	id, token, found := strings.Cut(strings.TrimSpace(connection), ":")
	data, decodeErr := base64.RawURLEncoding.Strict().DecodeString(token)
	if !found || !domain.ValidID(id) || decodeErr != nil || len(data) != 32 || len(token) != 43 {
		return State{}, errors.New("paste the connection code from the TestStudio recording page")
	}
	device, err := randomToken()
	if err != nil {
		return State{}, err
	}
	instance, err := randomToken()
	if err != nil {
		return State{}, err
	}
	return State{ID: id, Server: origin, Instance: "desktop-" + instance, DeviceToken: device, PairingToken: token}, nil
}

type Journal struct{ directory string }

const journalLimit = 20*recording.ContentLimit + 32768

func NewJournal(directory string) (*Journal, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, errors.New("private recording directory is unavailable")
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("recording directory must be private and must not be a symbolic link")
	}
	return &Journal{directory: directory}, nil
}

func (j *Journal) Read(id string) (State, error) {
	var state State
	if !domain.ValidID(id) {
		return state, errors.New("invalid recording session identifier")
	}
	path := filepath.Join(j.directory, id+".json")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > journalLimit {
		return state, errors.New("private recording connection is missing or unsafe")
	}
	file, err := os.Open(path)
	if err != nil {
		return state, errors.New("private recording connection is unavailable")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, journalLimit+1))
	if err != nil || len(data) > journalLimit || json.Unmarshal(data, &state) != nil || state.ID != id {
		return State{}, errors.New("private recording connection is invalid")
	}
	return state, nil
}

func (j *Journal) Create(state State) error {
	if !domain.ValidID(state.ID) {
		return errors.New("invalid recording session identifier")
	}
	data, err := json.Marshal(state)
	if err != nil {
		return errors.New("private recording connection cannot be saved")
	}
	file, err := os.OpenFile(filepath.Join(j.directory, state.ID+".json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return errors.New("recording connection already exists; resume it instead of pairing again")
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return errors.New("private recording connection cannot be saved")
	}
	if err := file.Sync(); err != nil {
		return errors.New("private recording connection cannot be saved")
	}
	return j.sync()
}

func (j *Journal) Save(state State) error {
	if !domain.ValidID(state.ID) {
		return errors.New("invalid recording session identifier")
	}
	path := filepath.Join(j.directory, state.ID+".json")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return errors.New("private recording connection is missing or unsafe")
	}
	data, err := json.Marshal(state)
	if err != nil {
		return errors.New("private recording connection cannot be saved")
	}
	file, err := os.CreateTemp(j.directory, ".connection-*")
	if err != nil {
		return errors.New("private recording connection cannot be saved")
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return errors.New("private recording connection cannot be saved")
	}
	if err := file.Sync(); err != nil {
		return errors.New("private recording connection cannot be saved")
	}
	if err := file.Close(); err != nil {
		return errors.New("private recording connection cannot be saved")
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return errors.New("private recording connection cannot be saved")
	}
	return j.sync()
}

func (j *Journal) sync() error {
	directory, err := os.Open(j.directory)
	if err != nil {
		return errors.New("private recording connection cannot be saved")
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return errors.New("private recording connection cannot be saved")
	}
	return nil
}
