// scan the given path and find the duplicated file
package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	MODE_DEBUG  = "debug"
	MODE_DELETE = "delete"
	MODE_BACKUP = "backup"
)

var flagPath = flag.String("path", "", "the duplication scan path")
var flagMode = flag.String("mode", "", "debug:show duplication only, delete: delete duplication file, backup: backup the duplication file to other path")
var flagBackupPath = flag.String("backup_path", "", "for mode=backup, backup the duplication file to this given backup path")

func initFlag() bool {
	flag.Parse()

	if *flagMode == "" {
		fmt.Println("mode is missing")
		flag.Usage()
		return false
	}

	if *flagPath == "" {
		fmt.Println("path is missing")
		flag.Usage()
		return false
	}

	if *flagMode != MODE_DEBUG && *flagMode != MODE_DELETE && *flagMode != MODE_BACKUP {
		fmt.Println("mode is invalid")
		flag.Usage()
		return false
	}

	if *flagMode == MODE_BACKUP && *flagBackupPath == "" {
		fmt.Println("mode=backup, backup_path is required")
		flag.Usage()
		return false
	}

	if !strings.HasSuffix(*flagBackupPath, "/") {
		*flagBackupPath = *flagBackupPath + "/"
	}

	return true
}

func main() {

	if !initFlag() {
		return
	}

	walk(load(*flagPath))
}

// deal wit the duplicated file
func dealWithDupFile(f img) {
	format := "[%s][%s] %s [%s]\n"
	switch *flagMode {
	case MODE_DEBUG:
		fmt.Printf(format, "-", f.FileInfo.ModTime(), f.FilePath, "debug only")

	case MODE_DELETE:
		if err := os.Remove(f.FilePath); err != nil {
			fmt.Printf(format, "X", f.FileInfo.ModTime(), f.FilePath, err.Error())
		} else {
			fmt.Printf(format, "X", f.FileInfo.ModTime(), f.FilePath, "Deleted")
		}

	case MODE_BACKUP:
		if err := os.Rename(f.FilePath, *flagBackupPath+filepath.Base(f.FilePath)); err != nil {
			fmt.Printf(format, "M", f.FileInfo.ModTime(), f.FilePath, err.Error())
		} else {
			fmt.Printf(format, "M", f.FileInfo.ModTime(), f.FilePath, "Backup")
		}
	}

}

// deal with the file we want to keep
func dealWithKeepFile(f img) {
	fmt.Printf("[O][%s] %s\n", f.FileInfo.ModTime(), f.FilePath)
}

// walk for those same size file, if their md5 value is the same,
// then keep the oldest one because the new one should be the copied one
func walk(im imgMap) {
	for _, imgs := range im {

		// only one image, nothing to do, just keep it
		if len(imgs) == 1 {
			continue
		}

		// calculate md5 into groups
		md5s := make(imgMap)
		for _, v := range imgs {
			md5str, err := md5File(v.FilePath)
			if err != nil {
				continue
			}

			md5s[md5str] = append(md5s[md5str], v)
		}

		for k, v := range md5s {
			if len(v) < 2 {
				continue
			}

			fmt.Printf("md5 %s\n", k)

			var keepFile img
			for _, f := range v {
				if keepFile.FilePath == "" {
					keepFile = f
				} else {
					if keepFile.FileInfo.ModTime().UTC().Unix() > f.FileInfo.ModTime().UTC().Unix() {
						dealWithDupFile(keepFile)
						keepFile = f
					} else {
						dealWithDupFile(f)
					}
				}
			}

			dealWithKeepFile(keepFile)
			fmt.Println("")
		}
	}
}

type img struct {
	FilePath string
	FileInfo os.FileInfo
}

// key: md5 value
// key: timestramp value
type imgMap map[string][]img

func (im *imgMap) add(key string, image ...img) {
	_, ok := (*im)[key]
	if !ok {
		(*im)[key] = make([]img, 0)
	}

	(*im)[key] = append((*im)[key], image...)
}

// scan the files with the same file size
func load(path string) imgMap {
	im := make(imgMap)

	err := filepath.Walk(path, func(subpath string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		if path == subpath || info.Name() == "." || info.Name() == ".." {
			return nil
		}

		if !info.IsDir() {
			im.add(fmt.Sprintf("%d", info.Size()), img{
				FilePath: subpath,
				FileInfo: info,
			})
		}

		return nil
	})

	if err != nil {
		fmt.Println(err)
	}

	return im
}

func md5File(filepath string) (md5str string, err error) {
	hash := md5.New()
	f, err := os.Open(filepath)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
