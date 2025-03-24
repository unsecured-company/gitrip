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
	"unsecured.company/gitrip/internal/utils"
)

const DirPerm = 0755
const FilePerm = 0644
const ObjectFilesCntWrongThreshold = 10

type Repo struct {
	dumper              *Dumper
	Url                 *gourl.URL
	Dir                 string
	FilesFetch          chan string
	FilesQueue          *FetchQueue
	wgFetcher           sync.WaitGroup
	wgFetch             sync.WaitGroup
	finished            atomic.Bool
	objectFilesCnt      atomic.Uint32
	objectFilesCntWrong int
	objectFilesSkip     bool
}

func NewRepo(dumper *Dumper, urlP *gourl.URL) (rp *Repo, err error) {
	utils.AddUrlSuffix(urlP, PathRoot)
	dumper.app.Out.Logf("Running for url (%s)", urlP.String())

	rp = &Repo{
		dumper:     dumper,
		Url:        urlP,
		FilesFetch: make(chan string, ChanFetchSize),
		FilesQueue: NewFetchQueue(),
	}

	exists, err := rp.createRootDir(rp.dumper.app.Cfg.DwnDir, urlP)

	if err == nil && exists && !dumper.app.Cfg.Update {
		err = errors.New("directory exists and update is disabled")
	}

	if err != nil {
		return
	}

	rp.dumper.app.Out.Debugf("Starting %d fetchers", dumper.app.Cfg.DwnThreads)

	for i := dumper.app.Cfg.DwnThreads; i > 0; i-- {
		rp.wgFetcher.Add(1)
		go rp.runnerFetch(i)
	}

	return
}

func (rp *Repo) run() (err error) {
	indexItem, err := rp.detect()

	if err != nil {
		return
	}

	rp.addPaths([]string{PathHead})
	rp.addPaths(getPathsRef())
	rp.addPaths(getPathsCommon())

	countFromIndex, _ := rp.addReferencesFromIndex(indexItem)
	rp.dumper.app.Out.Log(rp.logMsgf("%s files in GIT Index file", utils.NumToUnderscores(countFromIndex)))
	go rp.progressPrinter()

	rp.queueToChannel()

	rp.finished.Store(true)
	rp.dumper.app.Out.Log(rp.logMsgf("Done, fetched %d items", rp.FilesQueue.DoneCnt()))

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
	for path := range rp.FilesFetch {
		rp.dumper.app.Out.Debug(rp.logMsgf("Fetcher [%d] %s START", id, path))
		_, err := rp.fetchPath(path)
		rp.dumper.app.Out.Debug(rp.logMsgf("Fetcher [%d] %s DONE - %v", id, path, err))
	}

	rp.wgFetcher.Done()

	return
}

func (rp *Repo) queueToChannel() {
	doItAgain := true
	changesCnt := rp.FilesQueue.ChangesCnt()

	for doItAgain {
		name, exists := rp.FilesQueue.Get()

		if exists {
			rp.FilesFetch <- name

			continue
		}

		time.Sleep(time.Second)
		rp.wgFetch.Wait()

		doItAgain = rp.FilesQueue.HasThingsToDo() || rp.FilesQueue.ChangesCnt() > changesCnt
		changesCnt = rp.FilesQueue.ChangesCnt()
	}

	close(rp.FilesFetch)
	rp.wgFetcher.Wait()

	return
}

func (rp *Repo) fetchPath(name string) (it *Item, err error) {
	rp.wgFetch.Add(1)
	it = NewItem(rp, name, true)

	it.fetch(rp)
	err = it.netFetchErr

	rp.FilesQueue.Done(it.fileName)

	if it.exists {
		errRef := rp.getReferences(it)

		if errRef != nil {
			err = errors.Join(errRef, err)
		}

		rp.dumper.chanSave <- it
	}

	rp.wgFetch.Done()

	return
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
	indexItem = NewItem(rp, "index", true)
	indexItem.fetch(rp)
	hasIndex = len(indexItem.fileData) >= 5 && strings.HasPrefix(string(indexItem.fileData[:5]), "DIRC")

	return
}

func (rp *Repo) getReferences(it *Item) (err error) {
	hashes, err := rp.getReferencesFromData(it)
	rp.dumper.app.Out.Debugf(rp.logMsgf("References for %s | %d bytes : %v", it.fileName, it.fileSize, hashes))

	for _, hash := range hashes {
		rp.addHashToFetcher(hash)
	}

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
		rp.dumper.app.Out.Logf("Error getting references for %s: %v", it.fileName, err)

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

func (rp *Repo) addPath(name string) {
	rp.FilesQueue.Add(name)

	if isObjectFile(name) {
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
		rp.dumper.app.Out.Logf("Can not read index file: %v", err)

		return
	}

	for _, e := range index.Index.Entries {
		rp.dumper.app.Out.Debugf("adding hash from index %s", e.Hash.String())
		rp.addHashToFetcher(e.Hash.String())
		count++
	}

	return
}

func (rp *Repo) progressPrinter() {
	interval := time.Second * 2

	if rp.dumper.app.Cfg.Verbose {
		interval = time.Second * 2
	}

	for !rp.finished.Load() {
		rp.dumper.app.Out.Logf("%s downloaded %d, queue %d", rp.Url, rp.FilesQueue.DoneCnt(), rp.FilesQueue.TodoCnt())
		time.Sleep(interval)
	}
}

func (rp *Repo) checkObjectFileSkipping() {
	if rp.objectFilesSkip {
		return
	}

	if rp.objectFilesCntWrong >= ObjectFilesCntWrongThreshold && (int(rp.objectFilesCnt.Load())/10) < rp.objectFilesCntWrong {
		rp.dumper.app.Out.Log(rp.logMsg("Skipping /object/ files as lot of them are bad"))

		rp.objectFilesSkip = true
	}
}

func (rp *Repo) logMsg(msg string) string {
	return rp.Url.String() + " " + msg
}

func (rp *Repo) logMsgf(msg string, v ...interface{}) string {
	return fmt.Sprintf(rp.Url.String()+" "+msg, v...)
}

func isObjectFile(name string) bool {
	return strings.HasPrefix(name, "objects/")
}
