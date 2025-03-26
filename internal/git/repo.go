package git

import (
	"bytes"
	"errors"
	"fmt"
	gourl "net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsecured.company/gitrip/internal/application"
	"unsecured.company/gitrip/internal/utils"
)

const (
	DirPerm                      = 0755
	FilePerm                     = 0644
	ObjectFilesCntWrongThreshold = 10
	ProgressEvery                = time.Second * 2
)

type Repo struct {
	dumper              *Dumper
	cfg                 *application.Config
	out                 *application.Output
	Url                 *gourl.URL
	Dir                 string
	FilesQueue          *FetchQueue
	wgFetcher           sync.WaitGroup
	wgFetch             sync.WaitGroup
	wgFileProcess       sync.WaitGroup
	finished            atomic.Bool
	objectFilesCnt      atomic.Uint32
	objectFilesCntWrong int
	objectFilesSkip     bool
}

func NewRepo(dumper *Dumper, urlP *gourl.URL) (rp *Repo) {
	utils.AddUrlSuffix(urlP, PathRoot)

	return &Repo{
		dumper:     dumper,
		cfg:        dumper.app.Cfg,
		out:        dumper.app.Out,
		Url:        urlP,
		FilesQueue: NewFetchQueue(),
	}
}

func (rp *Repo) Run() (err error) {
	rp.out.Logf("Running for url (%s)", rp.Url.String())

	indexItem, err := rp.detect()
	if err != nil {
		return
	}

	exists, err := rp.createRootDir(rp.cfg.DwnDir, rp.Url)
	if err == nil && exists && !rp.cfg.Update {
		err = errors.New("directory exists and update is disabled")
	}

	if err != nil {
		return
	}

	rp.out.Debugf("Starting %d fetchers", rp.cfg.DwnThreads)
	for i := rp.cfg.DwnThreads; i > 0; i-- {
		rp.wgFetcher.Add(1)
		go rp.runnerFetch(i)
	}

	rp.addPaths(getPathsCommon())
	countFromIndex, _ := rp.addReferencesFromIndex(indexItem)
	rp.logf("%s files in GIT Index file", utils.NumToUnderscores(countFromIndex))

	go rp.progressPrinter()

	rp.out.Debugf("(%s) Waiting", rp.Url)
	rp.Wait()
	rp.finished.Store(true)
	rp.logf("done with %d items", rp.FilesQueue.CntDone())

	return
}

func (rp *Repo) createRootDir(dirBase string, url *gourl.URL) (exists bool, err error) {
	var dir string
	suffix := string(filepath.Separator) + PathRoot
	dirUrl := strings.TrimRight(url.String(), string(filepath.Separator))

	if strings.HasSuffix(dirUrl, suffix) {
		dirUrl = strings.TrimSuffix(dirUrl, suffix)
	}

	dirUrl, err = utils.UrlStrToFolderName(dirUrl)

	if err != nil {
		return
	}

	dir = filepath.Join(dirBase, dirUrl, PathRoot)
	_, err = os.Stat(dir)

	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(dir, DirPerm)
	} else {
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
	urlItem := utils.GetNewSuffixedUrl(rp.Url, path)
	data, httpCode, err := rp.dumper.fetcher.Fetch(rp.dumper.app.Ctx, urlItem.String())

	it = NewItem(rp, path, true)
	it.Update(data, httpCode, err)
	rp.FilesQueue.MarkDone(it.fileName)

	if it.exists {
		rp.wgFileProcess.Add(1)
		go rp.processFile(it)
	}

	rp.wgFetch.Done()

	return
}

func (rp *Repo) processFile(item *Item) {
	hashes, err := rp.getReferencesFromData(item)
	rp.out.Debugf("(%s) references for [%s] (%d bytes): %v", rp.Url, item.fileName, item.fileSize, hashes)

	for _, hash := range hashes {
		rp.addHashToFetcher(hash)
	}

	if err != nil {
		rp.logf("[%s] error getting references: %v", item.fileName, err)
	}

	rp.dumper.chanSave <- item
	rp.wgFileProcess.Done()
}

func (rp *Repo) detect() (indexItem *Item, err error) {
	hasIndex, indexItem, err := rp.hasIndexFile()

	if !hasIndex {
		err = errors.Join(errors.New("Not a valid GIT repository - .git/index file is missing or invalid."), err)
	}

	rp.dumper.chanSave <- indexItem

	return
}

func (rp *Repo) hasIndexFile() (hasIndex bool, indexItem *Item, err error) {
	urlItem := utils.GetNewSuffixedUrl(rp.Url, PathIndex)
	data, httpCode, err := rp.dumper.fetcher.Fetch(rp.dumper.app.Ctx, urlItem.String())

	indexItem = NewItem(rp, PathIndex, true)
	indexItem.Update(data, httpCode, err)
	hasIndex = indexItem.IsValidIndexFile()

	return
}

func (rp *Repo) addPaths(paths []string) {
	for _, path := range paths {
		rp.addPath(path)
	}
}

func (rp *Repo) addHashToFetcher(hash string) {
	name := rp.hashToPath(hash)

	rp.addPath(name)
}

func (rp *Repo) getReferencesFromData(it *Item) (hashes []string, err error) {
	if it.isObject && rp.objectFilesSkip {
		return
	}

	if it.fileName == PathHead {
		ref, isHash := it.getRefFromHead()

		if isHash {
			rp.addHashToFetcher(ref)
		} else {
			rp.addPath(ref)
			rp.addPath("logs/" + ref)
		}

		return
	}

	hashes, err = it.GetReferences()

	if err != nil {
		rp.out.Logf("Error getting references for %s: %v", it.fileName, err)

		if it.isObject {
			rp.objectFilesCntWrong++
			rp.checkObjectFileSkipping()
		}
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
		rp.objectFilesCnt.Add(1)
	}
}

func (rp *Repo) findHashes(data []byte) (hashes []string) {
	return rp.dumper.hashRegexp.FindAllString(string(data), -1)
}

func (rp *Repo) hashToPath(hash string) (path string) {
	return fmt.Sprintf("objects/%s/%s", hash[:2], hash[2:])
}

func (rp *Repo) hasItemChanCountBetween(xan chan *Item, defaultSize int) bool {
	cnt := len(xan)

	return cnt > 0 && cnt < defaultSize
}

func (rp *Repo) addReferencesFromIndex(it *Item) (count int, err error) {
	fr := bytes.NewReader(it.fileData)
	index, err := NewIndexFromReader(fr)

	if err != nil {
		rp.out.Logf("Can not read index file: %v", err)

		return
	}

	for _, e := range index.Index.Entries {
		//rp.out.Debugf("adding hash from index %s", e.Hash.String()) // TODO
		rp.addHashToFetcher(e.Hash.String())
		count++
	}

	return
}

func (rp *Repo) progressPrinter() {
	for !rp.finished.Load() {
		rp.logf("queued/done, %d/%d", rp.FilesQueue.CntQueued(), rp.FilesQueue.CntDone())
		time.Sleep(ProgressEvery)
	}
}

func (rp *Repo) checkObjectFileSkipping() {
	if rp.objectFilesSkip {
		return
	}

	if rp.objectFilesCntWrong >= ObjectFilesCntWrongThreshold && (int(rp.objectFilesCnt.Load())/10) < rp.objectFilesCntWrong {
		rp.out.Log(rp.logMsg("Skipping /object/ files as lot of them are bad"))

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
