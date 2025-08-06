package git

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/unsecured-company/gitrip/internal/application"
	"github.com/unsecured-company/gitrip/internal/utils"
)

const (
	DirPerm                  = 0755
	FilePerm                 = 0644
	ThresholdWrongObjectsPct = 50
	ThresholdWrongObjectsCnt = 200
	ProgressEveryXSec        = 2
)

type Repo struct {
	dumper            *Dumper
	cfg               *application.Config
	out               *application.Output
	Url               *url.URL
	Dir               string
	FilesQueue        *FetchQueue
	wgFetcher         sync.WaitGroup
	wgFetch           sync.WaitGroup
	wgFileProcess     sync.WaitGroup
	finished          atomic.Bool
	objectFilesCntAll atomic.Uint32
	objectFilesCntBad atomic.Uint32
	objectFilesSkip   bool
	regexpHash        *regexp.Regexp
}

func NewRepo(dumper *Dumper, urlP *url.URL) (rp *Repo) {
	utils.AddUrlSuffix(urlP, PathRoot)

	return &Repo{
		dumper:     dumper,
		cfg:        dumper.app.Cfg,
		out:        dumper.app.Out,
		Url:        urlP,
		FilesQueue: NewFetchQueue(),
		regexpHash: regexp.MustCompile(HashRegexp),
	}
}

func (rp *Repo) Run() (err error) {
	rp.out.Logf("(%s) Starting", rp.Url.String())
	indexItem, err := rp.detectAndStart()

	if err != nil {
		return
	}

	rp.out.Debugf("Starting %d fetchers", rp.cfg.DwnThreads)
	for i := rp.cfg.DwnThreads; i > 0; i-- {
		rp.wgFetcher.Add(1)
		go rp.runnerFetch(i)
	}

	go rp.progressPrinter()
	rp.addPaths(getPathsCommon())

	paths, err := indexItem.getPathFromIndexFile()

	if err == nil {
		rp.addPaths(paths)
		rp.logf("%s files in GIT Index file", utils.NumToUnderscores(len(paths)))
	}

	rp.out.Debugf("(%s) Waiting", rp.Url)
	rp.Wait()
	rp.finished.Store(true)
	rp.logf("done with %d items", rp.FilesQueue.CntDone())

	return
}

func (rp *Repo) setRootDir(dirBase string, urlP *url.URL) (exists bool, err error) {
	var dir string
	suffix := string(filepath.Separator) + PathRoot
	dirUrl := strings.TrimRight(urlP.String(), string(filepath.Separator))

	if strings.HasSuffix(dirUrl, suffix) {
		dirUrl = strings.TrimSuffix(dirUrl, suffix)
	}

	dirUrl, err = utils.UrlStrToFolderName(dirUrl)

	if err != nil {
		return
	}

	dir = filepath.Join(dirBase, dirUrl, PathRoot)
	_, err = os.Stat(dir)

	if err == nil || !os.IsNotExist(err) {
		exists = true
	}

	rp.Dir = dir

	return
}

func (rp *Repo) runnerFetch(id int) {
	for path := range rp.FilesQueue.Todo() {
		if application.DebugPrintEveryFetch {
			rp.out.Debug(rp.logMsgf("Fetcher [%d] %s START", id, path))
		}

		_, err := rp.fetchPath(path)

		if application.DebugPrintEveryFetch {
			rp.out.Debug(rp.logMsgf("Fetcher [%d] %s DONE - %v", id, path, err))
		}
	}

	rp.wgFetcher.Done()

	return
}

func (rp *Repo) fetchPath(path string) (it *Item, err error) {
	rp.wgFetch.Add(1)
	defer rp.wgFetch.Done()

	urlItem := utils.GetNewSuffixedUrl(rp.Url, path)
	data, httpCode, err := rp.dumper.fetcher.Fetch(rp.dumper.app.Ctx, urlItem.String(), 4)
	rp.FilesQueue.MarkDone(path)

	if httpCode >= 300 {
		return nil, fmt.Errorf("Non success code")
	}

	it = NewItem(rp.Dir, path, true, rp.out)
	it.Update(data, httpCode, err)

	if it.exists {
		rp.wgFileProcess.Add(1)
		go rp.processFile(it)
	}

	return
}

func (rp *Repo) processFile(item *Item) {
	paths, err := rp.getPathsFromData(item)
	rp.addPaths(paths)
	rp.out.Debugf("(%s)[%d] references for [%s] [%s] (%d bytes): %v", rp.Url, item.netHttpCode, item.fileName, item.objectType, item.fileSize, paths)

	if err != nil && !item.isObject {
		rp.logf("[%s] error getting references: %v", item.fileName, err)
	}

	rp.dumper.chanSave <- item
	rp.wgFileProcess.Done()
}

func (rp *Repo) detectAndStart() (indexItem *Item, err error) {
	exists, err := rp.setRootDir(rp.cfg.DwnDir, rp.Url)
	if err == nil && exists && !rp.cfg.Update {
		return indexItem, errors.New("directory exists and update is disabled")
	}

	hasIndex, indexItem, err := rp.hasIndexFile()
	if !hasIndex {
		err = errors.Join(errors.New("Not a valid GIT repository - .git/index file is missing or invalid."), err)
	}

	if err != nil {
		return
	}

	rp.dumper.chanSave <- indexItem

	return
}

func (rp *Repo) hasIndexFile() (hasIndex bool, indexItem *Item, err error) {
	rp.out.Debugf("(%s) checking for index file", rp.Url)
	urlItem := utils.GetNewSuffixedUrl(rp.Url, PathIndex)
	data, httpCode, err := rp.dumper.fetcher.Fetch(rp.dumper.app.Ctx, urlItem.String(), 4)
	rp.out.Debugf("(%s) check done, err: %v", rp.Url, err)

	indexItem = NewItem(rp.Dir, PathIndex, true, rp.out)
	indexItem.Update(data, httpCode, err)
	hasIndex = indexItem.IsValidIndexFile()

	return
}

func (rp *Repo) addPaths(paths map[string]bool) {
	for path, _ := range paths {
		rp.addPath(path)
	}
}

func (rp *Repo) getPathsFromData(it *Item) (paths map[string]bool, err error) {
	if it.isObject && rp.objectFilesSkip {
		return
	}

	paths, err = it.GetPaths()

	if err != nil && it.isObject {
		rp.objectFilesCntBad.Add(1)
		rp.checkObjectFileSkipping()
	}

	return
}

func readUntilByte(reader *bytes.Reader, delim byte) ([]byte, error) {
	var result []byte

	for {
		b, err := reader.ReadByte()

		if err != nil {
			return nil, err
		}

		if b == delim {
			break
		}

		result = append(result, b)
	}

	return result, nil
}

func (rp *Repo) addPath(path string) {
	rp.FilesQueue.Add(path)

	if isObjectFile(path) {
		rp.objectFilesCntAll.Add(1)
	}
}

func (rp *Repo) findHashes(data []byte) (hashes []string) {
	return rp.dumper.hashRegexp.FindAllString(string(data), -1)
}

func (rp *Repo) hasItemChanCountBetween(xan chan *Item, defaultSize int) bool {
	cnt := len(xan)

	return cnt > 0 && cnt < defaultSize
}

func (rp *Repo) progressPrinter() {
	for !rp.finished.Load() {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		msgMem := "Mem MB allocated/total/system/garbage %v/%v/%v/%v"
		msgMem = fmt.Sprintf(msgMem, m.Alloc/1024/1024, m.TotalAlloc/1024/1024, m.Sys/1024/1024, m.NumGC)

		rp.logf("queued/done, %d/%d [%d] %s", rp.FilesQueue.CntQueued(), rp.FilesQueue.CntDone(), len(rp.FilesQueue.todo), msgMem)
		time.Sleep(ProgressEveryXSec * time.Second)
	}
}

func (rp *Repo) checkObjectFileSkipping() {
	if rp.objectFilesSkip {
		return
	}

	cntAll := rp.objectFilesCntAll.Load()
	cntBad := rp.objectFilesCntBad.Load()
	isOverAbsoluteLimit := cntBad >= ThresholdWrongObjectsCnt
	isOverPercentage := cntBad*100 > cntAll*ThresholdWrongObjectsPct

	if isOverAbsoluteLimit && isOverPercentage {
		rp.logf("Skipping object files, %d/%d wrong", cntBad, cntAll)
		rp.objectFilesSkip = true
	}
}

func (rp *Repo) logMsg(msg string) string {
	return rp.Url.String() + " " + msg
}

func (rp *Repo) logMsgf(msg string, v ...interface{}) string {
	return fmt.Sprintf(rp.Url.String()+" "+msg, v...)
}

func (rp *Repo) logf(msg string, v ...interface{}) {
	rp.out.Logf(rp.Url.String()+" "+msg, v...)
}

func (rp *Repo) Wait() {
	for {
		rp.wgFileProcess.Wait()
		if rp.FilesQueue.CountersEqual() {
			time.Sleep(time.Millisecond * 200)
			//Trust but verify
			rp.wgFileProcess.Wait()
			if rp.FilesQueue.CountersEqual() {
				break
			}
		}

		time.Sleep(time.Millisecond * 200)
	}

	rp.FilesQueue.Close()
	rp.wgFetcher.Wait()
}

func isObjectFile(name string) bool {
	return strings.HasPrefix(name, "objects/")
}
