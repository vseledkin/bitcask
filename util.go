package bitcask

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	lockFileName    = "bitcask.lock"
	mergeDataSuffix = "merge.data"
	mergeHintSuffix = "merge.hint"
)

func getMergeHintFile(bc *BitCask) string {
	dirFp, err := os.OpenFile(bc.dirFile, os.O_RDONLY, 0755)
	if err != nil {
		panic(err)
	}
	defer dirFp.Close()
	fileLists, err := dirFp.Readdirnames(-1)
	if err != nil {
		panic(err)
	}

	for _, file := range fileLists {
		if strings.HasSuffix(file, mergeHintSuffix) {
			return file
		}
	}
	return bc.dirFile + "/" + strconv.Itoa(int(time.Now().Unix())) + mergeHintSuffix
}

func getMergeDataFile(bc *BitCask) string {
	dirFp, err := os.OpenFile(bc.dirFile, os.O_RDONLY, 0755)
	if err != nil {
		panic(err)
	}
	defer dirFp.Close()
	fileLists, err := dirFp.Readdirnames(-1)
	if err != nil {
		panic(err)
	}

	for _, file := range fileLists {
		if strings.HasSuffix(file, mergeDataSuffix) {
			return file
		}
	}
	return bc.dirFile + "/" + strconv.Itoa(int(time.Now().Unix())) + mergeDataSuffix
}

func checkWriteableFile(bc *BitCask) {
	if bc.writeFile.writeOffset > bc.Opts.MaxFileSize && bc.writeFile.fileID != uint32(time.Now().Unix()) {
		//logger.Info("open a new data/hint file:", bc.writeFile.writeOffset, bc.Opts.maxFileSize)
		//close data/hint fp
		bc.writeFile.hintFp.Close()
		bc.writeFile.fp.Close()

		writeFp, fileID := setWriteableFile(0, bc.dirFile)
		hintFp := setHintFile(fileID, bc.dirFile)
		bf := &BFile{
			fp:          writeFp,
			fileID:      fileID,
			writeOffset: 0,
			hintFp:      hintFp,
		}
		bc.writeFile = bf
		// update pid
		writePID(bc.lockFile, fileID)
	}
}

func listHintFiles(bc *BitCask) ([]string, error) {
	filterFiles := []string{mergeDataSuffix, mergeHintSuffix, lockFileName}
	dirFp, err := os.OpenFile(bc.dirFile, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return nil, err
	}
	defer dirFp.Close()
	//
	lists, err := dirFp.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	var hintLists []string
	for _, v := range lists {
		if strings.Contains(v, "hint") && !existsSuffixs(filterFiles, v) {
			hintLists = append(hintLists, v)
		}
	}
	return hintLists, nil
}

func listDataFiles(bc *BitCask) ([]string, error) {
	filterFiles := []string{mergeDataSuffix, mergeHintSuffix, lockFileName}
	dirFp, err := os.OpenFile(bc.dirFile, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return nil, err
	}
	defer dirFp.Close()
	//
	lists, err := dirFp.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	var dataFileLists []string
	for _, v := range lists {
		if strings.Contains(v, ".data") && !existsSuffixs(filterFiles, v) {
			dataFileLists = append(dataFileLists, v)
		}
	}
	return dataFileLists, nil
}

func lockFile(fileName string) (*os.File, error) {
	return os.OpenFile(fileName, os.O_EXCL|os.O_CREATE|os.O_RDWR, os.ModePerm)
}

func existsSuffixs(suffixs []string, src string) (b bool) {
	for _, suffix := range suffixs {
		b = strings.HasSuffix(src, suffix)
	}
	return
}

func writePID(pidFp *os.File, fileID uint32) {
	pidFp.WriteAt([]byte(strconv.Itoa(os.Getpid())+"\t"+strconv.Itoa(int(fileID))+".data"), 0)
}

func lastFileInfo(files []*os.File) (uint32, *os.File) {
	if files == nil {
		return uint32(0), nil
	}
	lastFp := files[0]

	fileName := lastFp.Name()
	s := strings.LastIndex(fileName, "/") + 1
	e := strings.LastIndex(fileName, ".hint")
	idx, _ := strconv.Atoi(fileName[s:e])
	lastID := idx
	for i := 0; i < len(files); i++ {
		idxFp := files[i]
		fileName = idxFp.Name()
		s = strings.LastIndex(fileName, "/") + 1
		e = strings.LastIndex(fileName, ".hint")
		idx, _ = strconv.Atoi(fileName[s:e])
		if lastID < idx {
			lastID = idx
			lastFp = idxFp
		}
	}
	return uint32(lastID), lastFp
}

func closeReadHintFp(files []*os.File, fileID uint32) {
	for _, fp := range files {
		if !strings.Contains(fp.Name(), strconv.Itoa(int(fileID))) {
			fp.Close()
		}
	}
}

func setWriteableFile(fileID uint32, dirName string) (*os.File, uint32) {
	var fp *os.File
	var err error
	if fileID == 0 {
		fileID = uint32(time.Now().Unix())
	}
	fileName := dirName + "/" + strconv.Itoa(int(fileID)) + ".data"
	fp, err = os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		panic(err)
	}
	return fp, fileID
}

func setHintFile(fileID uint32, dirName string) *os.File {
	var fp *os.File
	var err error
	if fileID == 0 {
		fileID = uint32(time.Now().Unix())
	}
	fileName := dirName + "/" + strconv.Itoa(int(fileID)) + ".hint"
	fp, err = os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		panic(err)
	}
	return fp
}

func appendWriteFile(fp *os.File, buf []byte) (int, error) {
	stat, err := fp.Stat()
	if err != nil {
		return -1, err
	}

	return fp.WriteAt(buf, stat.Size())
}

// return a unique file name by timeStamp
func uniqueFileName(root, suffix string) string {
	for {
		tStamp := strconv.Itoa(int(time.Now().Unix()))
		_, err := os.Stat(root + "/" + tStamp + "." + suffix)
		if err != nil && os.IsNotExist(err) {
			return tStamp + "." + suffix
		}
		time.Sleep(time.Second)
	}
}
