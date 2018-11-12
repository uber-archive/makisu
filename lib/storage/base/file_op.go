package base

import (
	"fmt"
	"os"

	"github.com/uber/makisu/lib/storage/metadata"
)

type lockLevel int

const (
	// lockLevelPeek is used as parameeter to lockHelper to indicate the lock is for peek.
	_lockLevelPeek lockLevel = iota
	// lockLevelRead indicates lock for read.
	_lockLevelRead
	// lockLevelWrite indicates lock for read.
	_lockLevelWrite
)

// FileOp performs one file or metadata operation on FileStore, given a list of acceptable states.
type FileOp interface {
	AcceptState(state FileState) FileOp

	CreateFile(name string, createState FileState, len int64) error
	MoveFileFrom(name string, createState FileState, sourcePath string) error
	MoveFile(name string, goalState FileState) error
	LinkFileTo(name string, targetPath string) error
	DeleteFile(name string) error

	GetFilePath(name string) (string, error)
	GetFileStat(name string) (os.FileInfo, error)

	GetFileReader(name string) (FileReader, error)
	GetFileReadWriter(name string) (FileReadWriter, error)

	GetFileMetadata(name string, md metadata.Metadata) error
	SetFileMetadata(name string, md metadata.Metadata) (bool, error)
	SetFileMetadataAt(name string, md metadata.Metadata, b []byte, offset int64) (bool, error)
	GetOrSetFileMetadata(name string, md metadata.Metadata) error
	DeleteFileMetadata(name string, md metadata.Metadata) error
}

var _ FileOp = (*localFileOp)(nil)

// localFileOp is a short-lived obj that performs one file or metadata operation on local disk, given a list of
// acceptable states.
type localFileOp struct {
	s      *localFileStore
	states map[FileState]interface{} // Set of states that's acceptable for the operation
}

// NewLocalFileOp inits a new FileOp obj.
func NewLocalFileOp(s *localFileStore) FileOp {
	return &localFileOp{
		s:      s,
		states: make(map[FileState]interface{}),
	}
}

// AcceptState adds a new state to the acceptable states list.
func (op *localFileOp) AcceptState(state FileState) FileOp {
	op.states[state] = struct{}{}
	return op
}

// verifyStateHelper verifies file is in one of the acceptable states.
func (op *localFileOp) verifyStateHelper(name string, entry FileEntry) error {
	currState := entry.GetState()
	for state := range op.states {
		if currState == state {
			// File is in one of the acceptable states.
			return nil
		}
	}
	return &FileStateError{
		Op:    "verifyStateHelper",
		Name:  name,
		State: currState,
		Msg:   fmt.Sprintf("desired states: %v", op.states),
	}
}

// reloadFileEntryHelper tries to reload file from disk into memory.
// Note it doesn't try to verify states or reload file from all possible states.
// If reload succeeded, return true;
// If file already exists in memory, return false;
// If file is neither in memory or on disk, return false with os.ErrNotExist.
// TODO: If file doesn't exist on disk, this function would still get a entry lock just to verify.
// This might block actual file creation.
func (op *localFileOp) reloadFileEntryHelper(name string) (reloaded bool, err error) {
	if op.s.fileMap.Contains(name) {
		return false, nil
	}

	// Check if file exists on disk.
	for state := range op.states {
		fileEntry := op.s.fileEntryFactory.Create(name, state)
		// Try load before acquiring lock first.
		if err = fileEntry.Reload(); err != nil {
			continue
		}
		// Try to store file entry into memory.
		// It's possible the entry was just reloaded by another goroutine before this point, then
		// false will be returned.
		// It's also possible the entry was just added/reloaded and then deleted, in which case
		// os.ErrNotExist will be returned, and the newly added file entry will be deleted from map.
		_, loaded := op.s.fileMap.LoadOrStore(
			name, fileEntry, func(name string, entry FileEntry) error {
				// Verify the file is still on disk.
				err = entry.Reload()
				return err
			})
		if loaded {
			return false, nil
		} else if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, os.ErrNotExist
}

// lockHelper runs f under protection of entry level RWMutex.
func (op *localFileOp) lockHelper(
	name string, l lockLevel, f func(name string, entry FileEntry)) (err error) {
	if _, err = op.reloadFileEntryHelper(name); err != nil {
		return err
	}
	var loaded bool
	if l == _lockLevelPeek {
		loaded = op.s.fileMap.LoadForPeek(name, func(name string, entry FileEntry) {
			if err = op.verifyStateHelper(name, entry); err != nil {
				return
			}
			f(name, entry)
		})
	} else if l == _lockLevelRead {
		loaded = op.s.fileMap.LoadForRead(name, func(name string, entry FileEntry) {
			if err = op.verifyStateHelper(name, entry); err != nil {
				return
			}
			f(name, entry)
		})
	} else if l == _lockLevelWrite {
		loaded = op.s.fileMap.LoadForWrite(name, func(name string, entry FileEntry) {
			if err = op.verifyStateHelper(name, entry); err != nil {
				return
			}
			f(name, entry)
		})
	}
	if !loaded {
		return os.ErrNotExist
	}
	return err
}

func (op *localFileOp) deleteHelper(
	name string, f func(name string, entry FileEntry) error) (err error) {
	if _, err = op.reloadFileEntryHelper(name); err != nil {
		return err
	}
	op.s.fileMap.Delete(name, func(name string, entry FileEntry) error {
		err = op.verifyStateHelper(name, entry)
		if err != nil {
			return err
		}
		return f(name, entry)
	})
	return err
}

// createFileHelper is a helper function that adds a new file to store.
// it either moves the new file from a unmanaged location, or creates an empty file with specified size.
// If file exists and is in one of the acceptable states, returns os.ErrExist.
// If file exists but not in one of the acceptable states, returns FileStateError.
func (op *localFileOp) createFileHelper(
	name string, targetState FileState, sourcePath string, len int64) (err error) {
	// Check if file exists in in-memory map and is in an acceptable state.
	loaded := op.s.fileMap.LoadForRead(name, func(name string, entry FileEntry) {
		err = op.verifyStateHelper(name, entry)
	})
	if err != nil && !os.IsNotExist(err) {
		// Includes FileStateError.
		return err
	} else if loaded {
		return os.ErrExist
	}

	// Check if file is on disk.
	loaded, err = op.reloadFileEntryHelper(name)
	if err != nil && !os.IsNotExist(err) {
		// Includes FileStateError.
		return err
	} else if loaded {
		return os.ErrExist
	}

	// Create new file entry.
	err = nil
	newEntry := op.s.fileEntryFactory.Create(name, targetState)
	actual, loaded := op.s.fileMap.LoadOrStore(name, newEntry, func(name string, entry FileEntry) error {
		if sourcePath != "" {
			err = newEntry.MoveFrom(targetState, sourcePath)
			if err != nil {
				return err
			}
		} else {
			err = newEntry.Create(targetState, len)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	} else if loaded {
		// Another goroutine created the entry before this one.
		// Verify again for a correct error message.
		if err := op.verifyStateHelper(name, actual); err != nil {
			return err
		}
		return os.ErrExist
	}

	return nil
}

// CreateFile creates an empty file with specified size.
// If file exists and is in one of the acceptable states, returns os.ErrExist.
// If file exists but not in one of the acceptable states, returns FileStateError.
func (op *localFileOp) CreateFile(name string, targetState FileState, len int64) (err error) {
	return op.createFileHelper(name, targetState, "", len)
}

// MoveFileFrom moves an unmanaged file into file store.
// If file exists and is in one of the acceptable states, returns os.ErrExist.
// If file exists but not in one of the acceptable states, returns FileStateError.
func (op *localFileOp) MoveFileFrom(name string, targetState FileState, sourcePath string) (err error) {
	return op.createFileHelper(name, targetState, sourcePath, -1)
}

// MoveFile moves a file to a different directory and updates its state accordingly, and moves all metadata that's
// `movable`.
func (op *localFileOp) MoveFile(name string, targetState FileState) (err error) {
	if _, err = op.reloadFileEntryHelper(name); err != nil {
		return err
	}

	// Verify that the file is not in target state, and is currently in one of the acceptable states.
	loaded := op.s.fileMap.LoadForWrite(name, func(name string, entry FileEntry) {
		currState := entry.GetState()
		if currState == targetState {
			err = os.ErrExist
			return
		}
		for state := range op.states {
			if currState == state {
				// File is in one of the acceptable states. Perform move.
				err = entry.Move(targetState)
				return
			}
		}
		err = &FileStateError{
			Op:    "MoveFile",
			State: currState,
			Name:  name,
			Msg:   fmt.Sprintf("desired states: %v", op.states),
		}
	})
	if !loaded {
		return os.ErrNotExist
	}
	return err
}

// LinkFileTo create a hardlink to an unmanaged path.
func (op *localFileOp) LinkFileTo(name string, targetPath string) (err error) {
	if loadErr := op.lockHelper(name, _lockLevelRead, func(name string, entry FileEntry) {
		err = entry.LinkTo(targetPath)
	}); loadErr != nil {
		return loadErr
	}
	return err
}

// DeleteFile removes a file from disk and file map.
func (op *localFileOp) DeleteFile(name string) (err error) {
	if loadErr := op.deleteHelper(name, func(name string, entry FileEntry) error {
		err = entry.Delete()
		return nil
	}); loadErr != nil {
		return loadErr
	}
	return err
}

// GetFilePath returns full path for a file.
func (op *localFileOp) GetFilePath(name string) (path string, err error) {
	if loadErr := op.lockHelper(name, _lockLevelPeek, func(name string, entry FileEntry) {
		path = entry.GetPath()
	}); loadErr != nil {
		return "", loadErr
	}
	return path, nil
}

// GetFileStat returns FileInfo for a file.
func (op *localFileOp) GetFileStat(name string) (info os.FileInfo, err error) {
	if loadErr := op.lockHelper(name, _lockLevelPeek, func(name string, entry FileEntry) {
		info, err = entry.GetStat()
	}); loadErr != nil {
		return nil, loadErr
	}
	return info, err
}

// GetFileReader returns a FileReader object for read operations.
func (op *localFileOp) GetFileReader(name string) (r FileReader, err error) {
	if loadErr := op.lockHelper(name, _lockLevelRead, func(name string, entry FileEntry) {
		r, err = entry.GetReader()
	}); loadErr != nil {
		return nil, loadErr
	}
	return r, err
}

// GetFileReadWriter returns a FileReadWriter object for read/write operations.
func (op *localFileOp) GetFileReadWriter(name string) (w FileReadWriter, err error) {
	if loadErr := op.lockHelper(name, _lockLevelWrite, func(name string, entry FileEntry) {
		w, err = entry.GetReadWriter()
	}); loadErr != nil {
		return nil, loadErr
	}
	return w, err
}

// GetFileMetadata loads metadata assocciated with the file.
func (op *localFileOp) GetFileMetadata(name string, md metadata.Metadata) (err error) {
	if loadErr := op.lockHelper(name, _lockLevelPeek, func(name string, entry FileEntry) {
		err = entry.GetMetadata(md)
	}); loadErr != nil {
		return loadErr
	}
	return err
}

// SetFileMetadata creates or overwrites metadata assocciate with the file.
func (op *localFileOp) SetFileMetadata(name string, md metadata.Metadata) (updated bool, err error) {
	if loadErr := op.lockHelper(name, _lockLevelWrite, func(name string, entry FileEntry) {
		updated, err = entry.SetMetadata(md)
	}); loadErr != nil {
		return false, loadErr
	}
	return updated, err
}

// SetFileMetadataAt overwrites metadata assocciate with the file with content.
func (op *localFileOp) SetFileMetadataAt(
	name string, md metadata.Metadata, b []byte, offset int64) (updated bool, err error) {

	if loadErr := op.lockHelper(name, _lockLevelWrite, func(name string, entry FileEntry) {
		updated, err = entry.SetMetadataAt(md, b, offset)
	}); loadErr != nil {
		return false, loadErr
	}
	return updated, err
}

// GetOrSetFileMetadata see localFileEntryInternal.
func (op *localFileOp) GetOrSetFileMetadata(name string, md metadata.Metadata) (err error) {
	if loadErr := op.lockHelper(name, _lockLevelWrite, func(name string, entry FileEntry) {
		err = entry.GetOrSetMetadata(md)
	}); loadErr != nil {
		return loadErr
	}
	return err
}

// DeleteFileMetadata deletes metadata of the specified type for a file.
func (op *localFileOp) DeleteFileMetadata(name string, md metadata.Metadata) (err error) {
	loadErr := op.lockHelper(name, _lockLevelWrite, func(name string, entry FileEntry) {
		err = entry.DeleteMetadata(md)
	})
	if loadErr != nil {
		return loadErr
	}
	return err
}
