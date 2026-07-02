package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

//dirStats walks a directory path recusively and returns:
// total size of all files
// most recent modification time seen
// error if any

func dirStats(path string) (int64, time.Time, error){
	var totalSize int64
	var maxMod time.Time
	
	//walkdir recursively walks the dile tree root path
	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		//if there is and error accessing this specific file or folder return nil

		if err != nil {
			return nil
		}

		// only consider file size
		if d.IsDir() {
			return nil
		}

		// retrive detailed file info
		info, err := d.Info()
		if err != nil {
			return nil // skip if cant file details info
		}

		//add file size to running total
		totalSize += info.Size()

		// if file's modification time is newer tha the newest we have seen, update it
		if info.ModTime().After(maxMod){
			maxMod = info.ModTime()
		}
		
		return nil
	})

	return totalSize, maxMod, err
}

// single file equivalent of dirStats
func fileStats(path string) (int64, time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, time.Time{}, err
	}

	return info.Size(), info.ModTime(), nil
}