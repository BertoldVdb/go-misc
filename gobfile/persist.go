package gobpersist

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io/ioutil"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// GobPersist is a simple helper package that allows saving and loading a target to a file using gob.
// It also handles timed saves
type GobPersist struct {
	sync.RWMutex
	modified uint32

	// Filename is the name of the file to use to persist the information
	Filename string
	// Target needs to be set to a pointer to the structure or field that is to be persisted
	Target interface{}
	// SaveInterval is the minimum interval between conditional saves.
	SaveInterval time.Duration

	buffer  bytes.Buffer
	encoder *gob.Encoder
	decoder *gob.Decoder

	nextSave time.Time
}

const (
	// RetrySaveInterval is the delay between save attempts if the previous one failed.
	RetrySaveInterval = 2 * time.Second
)

var (
	// ErrorNoFilename is returned when trying to save without specifying a file
	ErrorNoFilename = errors.New("Filename not specified")
)

// Load will try to restore the structure Target points to.
func (g *GobPersist) Load() error {
	if g.Filename == "" {
		return ErrorNoFilename
	}

	g.Lock()
	defer g.Unlock()

	if g.decoder == nil {
		g.decoder = gob.NewDecoder(&g.buffer)
	}

	g.buffer.Truncate(0)
	file, err := os.Open(g.Filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = g.buffer.ReadFrom(file)
	if err == nil {
		err = g.decoder.Decode(g.Target)
	}

	return err
}

// Save will write the Target to file, regardless if it was changed or how long ago the previous save was.
func (g *GobPersist) Save() error {
	if g.Filename == "" {
		return nil
	}

	tmpName := g.Filename + ".tmp"

	g.RLock()
	defer g.RUnlock()

	if g.encoder == nil {
		g.encoder = gob.NewEncoder(&g.buffer)
	}

	g.buffer.Truncate(0)
	err := g.encoder.Encode(g.Target)
	if err != nil {
		goto done
	}

	err = ioutil.WriteFile(tmpName, g.buffer.Bytes(), 0600)
	if err != nil {
		goto done
	}

	err = os.Rename(tmpName, g.Filename)

done:
	if err == nil {
		g.nextSave = time.Now().Add(g.SaveInterval)
	} else {
		g.nextSave = time.Now().Add(RetrySaveInterval)
	}

	return err
}

// SaveConditional performs a save operation if the Target is modified and the minimum 'SaveInterval'
// has passed. If modified is true, Touch is called internally.
func (g *GobPersist) SaveConditional(modified bool) error {
	if g.Filename == "" {
		return nil
	}

	if modified {
		g.Touch()
	}

	var err error

	if atomic.CompareAndSwapUint32(&g.modified, 1, 0) {
		g.RLock()

		if time.Now().After(g.nextSave) {
			err = g.Save()
		}

		g.RUnlock()
	}

	return err
}

// Touch signals that the Target has been changed and should be called after modifications
func (g *GobPersist) Touch() {
	atomic.StoreUint32(&g.modified, 1)
}
