package rotatelogs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type rotateConfig struct {
	fileBasePath       string
	fileBaseName 	   string
	subDateDir 		   string
	suffixFileDateTimeLayout string

	maxAge time.Duration
	maxNum int

	logChannelLen        int
	rotationTime         time.Duration
	rotateLogFileMaxSize int64
	fileExt              string
}

type logFile struct {
	filePath string
	dateTime time.Time
}

type hisLogFile struct {
	files []logFile
}

type RotateLogs struct {
	rotateConfig
	hisLogFile

	currFileName string
	currFileDate time.Time
	currFileSize int64

	eventChannel chan bool

	writer         IWriter
	outFh          *os.File
	outMutex       sync.Mutex
	nextRotateTime time.Time
}

func (rl *RotateLogs) Write(p []byte) (int, error) {
	ok := rl.rotateFileTime()
	n, err := rl.writer.Write(p)
	if err != nil {
		return 0, err
	}

	if !ok {
		err = rl.rotateFileSize(len(p))
	}

	return n, err
}

func (rl *RotateLogs) getNextRotationTime() time.Time {
	nowTime := time.Now()
	zeroDate := time.Date(nowTime.Year(), nowTime.Month(), nowTime.Day(), 0, 0, 0, 0, time.Local)
	passTime := nowTime.Sub(zeroDate)

	passCnt := passTime/rl.rotationTime + 1
	return zeroDate.Add(passCnt * rl.rotationTime)
}

func (rl *RotateLogs) rotateFileTime() bool {
	if rl.rotationTime == 0 {
		return false
	}

	if time.Now().Before(rl.nextRotateTime) {
		return false
	}

	rl.nextRotateTime = rl.getNextRotationTime()
	return rl.rotateFile() == nil
}

func (rl *RotateLogs) rotateFileSize(size int) error {
	if rl.rotateLogFileMaxSize <= 0 {
		return nil
	}

	rl.currFileSize += int64(size)
	if rl.currFileSize < rl.rotateLogFileMaxSize {
		return nil
	}

	return rl.rotateFile()
}

func (rl *RotateLogs) setNewFD(fd *os.File, fileName string, fileSize int64) {
	if rl.outFh != nil {
		rl.outFh.Close()
		rl.appendNewFile(rl.currFileName)
	}

	rl.outFh = fd
	rl.currFileName = fileName
	rl.currFileSize = fileSize
}

func (rl *RotateLogs) rotateFile() error {
	fh, fileName, fileSize, err := rl.openNewFile()
	if err != nil {
		return err
	}

	rl.outMutex.Lock()
	defer rl.outMutex.Unlock()
	rl.setNewFD(fh, fileName, fileSize)

	return nil
}

func (rl *RotateLogs) writeToFile(p []byte) (int, error) {
	rl.outMutex.Lock()
	defer rl.outMutex.Unlock()

	return rl.outFh.Write(p)
}

func (rl *RotateLogs) Sync() error {
	err := rl.writer.Sync()
	if err != nil {
		return err
	}

	rl.outMutex.Lock()
	defer rl.outMutex.Unlock()
	return rl.outFh.Sync()
}

// Close satisfies the io.Closer interface. You must
// call this method if you performed any writes to
// the object.
func (rl *RotateLogs) Close() error {
	rl.writer.Close()
	rl.outMutex.Lock()
	defer rl.outMutex.Unlock()
	return rl.outFh.Close()
}

func (rl *RotateLogs) setDefaultConfig() {
	if rl.fileExt == "" {
		rl.fileExt = DefaultFileExt
	}
}

func (rl *RotateLogs) getFileName() string {
	return filepath.Join(rl.fileBasePath,time.Now().Format(rl.subDateDir),rl.fileBaseName+ time.Now().Format(rl.suffixFileDateTimeLayout)+rl.fileExt)
}

func (rl *RotateLogs) removeFile(files []string) {
	go func() {
		for _, file := range files {
			os.Remove(file)
		}
	}()
}

func (rl *RotateLogs) appendNewFile(fileName string) {
	if rl.maxNum == 0 && rl.maxAge == 0 {
		return
	}

	var removeFile []string
	rl.files = append(rl.files, logFile{filePath: fileName, dateTime: time.Now()})
	if rl.maxNum > 0 && len(rl.files) > rl.maxNum {
		removeNum := len(rl.files) - rl.maxNum
		for i := 0; i < removeNum; i++ {
			removeFile = append(removeFile, rl.files[i].filePath)
		}

		rl.files = rl.files[removeNum:]
	}

	if rl.maxAge > 0 {
		removeIdx := -1
		for i := range rl.files {
			if rl.files[i].dateTime.Add(rl.maxAge).Before(time.Now()) {
				removeFile = append(removeFile, rl.files[i].filePath)
				removeIdx = i
			} else {
				break
			}
		}
		rl.files = rl.files[removeIdx+1:]
	}

	rl.removeFile(removeFile)
}

func (rl *RotateLogs) walkLogs() {
	if rl.maxNum == 0 && rl.maxAge == 0 {
		return
	}
	// walk all file
	filepath.Walk(rl.fileBasePath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != rl.fileExt {
			return nil
		}

		rl.files = append(rl.files, logFile{
			filePath: path,
			dateTime: info.ModTime(),
		})

		return nil
	})

	// sort files by mod time
	sort.Slice(rl.files, func(i, j int) bool {
		return rl.files[i].dateTime.Before(rl.files[j].dateTime)
	})
}

func (rl *RotateLogs) prepare() error {
	rl.walkLogs()
	fh, fileName, size, err := rl.openNewFile()
	if err != nil {
		return err
	}

	rl.setNewFD(fh, fileName, size)
	rl.currFileDate = time.Now()
	if rl.rotationTime != 0 {
		rl.nextRotateTime = rl.getNextRotationTime()
		a := rl.nextRotateTime.Format("2006-01-02 15:04:05")
		fmt.Println(a)
	}

	return nil
}

func (rl *RotateLogs) openNewFile() (*os.File, string, int64, error) {
	// open the log file
	filePathName := rl.getFileName()
	dirName := filepath.Dir(filePathName)
	if err := os.MkdirAll(dirName, 0755); err != nil {
		return nil, "", 0, fmt.Errorf("failed to create directory %s", dirName)
	}

	fh, err := os.OpenFile(filePathName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to open file %ss", filePathName)
	}

	fInfo, err := fh.Stat()
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to stat file %ss", filePathName)
	}

	return fh, filePathName, fInfo.Size(), nil
}

func (rl *RotateLogs) isChangeDay() bool {
	return time.Now().Year() != rl.currFileDate.Year() ||
		time.Now().Month() != rl.currFileDate.Month() ||
		time.Now().Day() != rl.currFileDate.Day()
}

func NewRotateLogs(basePath string, baseFileName string,subDateDir string,suffixFileDateTimeLayout string, options ...Option) (*RotateLogs, error) {
	dir, err := os.Stat(basePath)
	if err != nil || dir.IsDir() == false {
		return nil, errors.New("Not found dir " + basePath)
	}

	rl := &RotateLogs{}
	rl.fileBasePath = basePath
	rl.fileBaseName = baseFileName
	rl.subDateDir = subDateDir
	rl.suffixFileDateTimeLayout = suffixFileDateTimeLayout

	err = checkFileNameDateTimeLayout(filepath.Base(suffixFileDateTimeLayout))
	if err != nil {
		return nil, err
	}

	rl.fileExt = filepath.Ext(suffixFileDateTimeLayout)
	rl.suffixFileDateTimeLayout = strings.TrimRight(suffixFileDateTimeLayout, rl.fileExt)

	for _, option := range options {
		option.Configure(rl)
	}

	rl.setDefaultConfig()
	err = rl.prepare()
	if err != nil {
		return nil, err
	}

	if rl.logChannelLen > 0 {
		rl.writer = newChannelWriter(rl.writeToFile, rl.logChannelLen)
	} else {
		rl.writer = newFileWriter(rl.writeToFile)
	}

	return rl, nil
}

// WithFileNameDateTimeLayout 设置文件格式
func checkFileNameDateTimeLayout(fileNameDateTimeLayout string) error {
	if strings.IndexAny(fileNameDateTimeLayout, "2006") == -1 {
		return fmt.Errorf("invalid date time layout")
	}
	if strings.IndexAny(fileNameDateTimeLayout, "01") == -1 {
		return fmt.Errorf("invalid date time layout")
	}

	if strings.IndexAny(fileNameDateTimeLayout, "02") == -1 {
		return fmt.Errorf("invalid date time layout")
	}

	if strings.IndexAny(fileNameDateTimeLayout, "15") == -1 {
		return fmt.Errorf("invalid date time layout")
	}

	if strings.IndexAny(fileNameDateTimeLayout, "04") == -1 {
		return fmt.Errorf("invalid date time layout")
	}
	if strings.IndexAny(fileNameDateTimeLayout, "05") == -1 {
		return fmt.Errorf("invalid date time layout")
	}

	return nil
}
