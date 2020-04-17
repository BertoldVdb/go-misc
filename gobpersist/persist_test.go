package gobpersist

import (
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func tempDir() string {
	dir, err := ioutil.TempDir(os.TempDir(), "test-")
	if err != nil {
		panic(err)
	}
	return dir
}

func setupGob(dir string, value interface{}) *GobPersist {
	return &GobPersist{
		Filename: dir + "/gob",
		Target:   value,
	}
}
func TestSaveLoad(t *testing.T) {
	dir := tempDir()
	defer os.RemoveAll(dir)

	value := 1
	gob := setupGob(dir, &value)

	/* Save multiple times */
	for i := 0; i < 10; i++ {
		if gob.Save() != nil {
			t.Error("Save failed")
		}
	}

	/* Load multiple times */
	for i := 0; i < 10; i++ {
		value = 0
		if gob.Load() != nil {
			t.Error("Load failed")
		}
	}

	if value != 1 {
		t.Error("Value was not saved and loaded")
	}
}

func TestSaveLoadNoFilename(t *testing.T) {
	value := 0
	gob := &GobPersist{
		Target: &value,
	}

	if gob.Save() != nil {
		t.Error("Save failed")
	}

	if gob.SaveConditional(false) != nil {
		t.Error("Save conditional failed")
	}

	if gob.Load() != ErrorNoFilename {
		t.Error("Load did not fail")
	}
}

func testSaveLoadConditional(modified bool, touch bool, sleep time.Duration, saveInterval time.Duration) int {
	dir := tempDir()
	defer os.RemoveAll(dir)

	value := 0
	gob := setupGob(dir, &value)
	gob.SaveInterval = saveInterval

	if gob.Save() != nil {
		return -1
	}

	value = 1

	if touch {
		gob.Touch()
	}
	time.Sleep(sleep)

	if gob.SaveConditional(modified) != nil {
		return -1
	}

	if gob.Load() != nil {
		return -1
	}

	return value
}

func TestSaveLoadConditional(t *testing.T) {
	/* Non time based */
	if testSaveLoadConditional(false, false, 0, 0) != 0 {
		t.Error("T1")
	}
	if testSaveLoadConditional(true, false, 0, 0) != 1 {
		t.Error("T2")
	}
	if testSaveLoadConditional(false, true, 0, 0) != 1 {
		t.Error("T3")
	}
	if testSaveLoadConditional(true, true, 0, 0) != 1 {
		t.Error("T4")
	}
	/* Time based */
	sleep := 10 * time.Millisecond
	longTime := 5 * sleep
	shortTime := sleep / 5

	if testSaveLoadConditional(true, false, sleep, shortTime) != 1 {
		t.Error("T5")
	}
	if testSaveLoadConditional(true, false, sleep, longTime) != 0 {
		t.Error("T6")
	}
}

func TestBadPathError(t *testing.T) {
	value := 0
	gob := setupGob("./this/path/does/not/exist", &value)

	if gob.Save() == nil {
		t.Error("Could save to non existing file")
	}

	delta := gob.nextSave.Sub(time.Now())
	if delta < RetrySaveInterval/2 || delta > 3*RetrySaveInterval/2 {
		t.Error("Next interval not correct")
	}

	/* Not a good test */
	if gob.Load() == nil {
		t.Error("Could load from bad file")
	}
}

func TestUnencodable(t *testing.T) {
	type noExportedFields struct {
		field int
	}

	dir := tempDir()
	defer os.RemoveAll(dir)

	value := noExportedFields{field: 1234}
	gob := setupGob(dir, &value)

	if gob.Save() == nil {
		t.Error("Gob errors not passed through")
	}
}
