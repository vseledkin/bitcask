package bitcask

import "sync"

// KeyDirs for HashMap
var keyDirsLock *sync.Mutex

var keyDirs *KeyDirs
var keyDirsOnce sync.Once

func init() {
	keyDirsLock = &sync.Mutex{}
}

// KeyDirs ...
type KeyDirs struct {
	entrys map[string]*entry
}

// NewKeyDir return a KeyDir Obj
func NewKeyDir(dirName string, timeoutSecs int) *KeyDirs {
	//filepath.Abs(fp.Name())
	keyDirsLock.Lock()
	defer keyDirsLock.Unlock()

	keyDirsOnce.Do(func() {
		if keyDirs == nil {
			keyDirs = &KeyDirs{}
		}
	})
	return keyDirs
}

func (keyDirs *KeyDirs) put(key string, e *entry) {
	keyDirsLock.Lock()
	defer keyDirsLock.Unlock()

	old, ok := keyDirs.entrys[key]
	if !ok || e.isNewerThan(old) {
		keyDirs.entrys[key] = e
		return
	}

	keyDirs.entrys[key] = old
}
