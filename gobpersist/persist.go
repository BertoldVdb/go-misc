package gobpersist

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

// GobPersist is a simple helper package that allows saving and loading a target to a file using gob.
// It also handles timed saves
type GobPersist struct {
	sync.Mutex
	modified bool

	// Filename is the name of the file to use to persist the information
	Filename string
	// Target needs to be set to a pointer to the structure or field that is to be persisted
	Target interface{}
	// SaveInterval is the minimum interval between conditional saves.
	SaveInterval time.Duration

	buffer bytes.Buffer

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
	g.Lock()
	defer g.Unlock()

	if g.Filename == "" {
		return ErrorNoFilename
	}

	g.buffer.Truncate(0)
	file, err := os.Open(g.Filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = g.buffer.ReadFrom(file)
	if err == nil {
		err = gob.NewDecoder(&g.buffer).Decode(g.Target)
	}

	return err
}

func (g *GobPersist) save() error {
	if g.Filename == "" {
		return nil
	}

	tmpName := g.Filename + ".tmp"

	g.buffer.Truncate(0)
	err := gob.NewEncoder(&g.buffer).Encode(g.Target)
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
		g.modified = false
		g.nextSave = time.Now().Add(g.SaveInterval)
	} else {
		g.nextSave = time.Now().Add(RetrySaveInterval)
	}

	return err
}

// Save will write the Target to file, regardless if it was changed or how long ago the previous save was.
func (g *GobPersist) Save() error {
	g.Lock()
	defer g.Unlock()

	return g.save()
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

	g.Lock()
	if g.modified && time.Now().After(g.nextSave) {
		err = g.save()
	}
	g.Unlock()

	return err
}

// Touch signals that the Target has been changed and should be called after modifications
func (g *GobPersist) Touch() {
	g.Lock()
	defer g.Unlock()

	g.modified = true
}
