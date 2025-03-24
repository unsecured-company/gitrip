package git

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unsafe"
	"unsecured.company/gitrip/internal/application"
	"unsecured.company/gitrip/internal/utils"
)

// TODO This is in general not too good.
// - Fetch should be done outside.
// - Better check for the data.

type Item struct {
	repo        *Repo
	doRefs      bool
	exists      bool
	isObject    bool
	fileData    []byte
	fileDataStr string
	fileName    string
	fileSize    int
	netFetchErr error
	netHttpCode int
}

func NewItem(rp *Repo, name string, doReferences bool) (i *Item) {
	i = &Item{}
	i.repo = rp
	i.doRefs = doReferences
	i.fileName = name
	i.isObject = strings.HasPrefix(name, "objects/")

	return
}

func (it *Item) save() (err error) {
	if it.exists == false {
		return
	}

	fp := filepath.Join(it.repo.Dir, it.fileName)
	dir := filepath.Dir(fp)
	err = os.MkdirAll(dir, DirPerm)

	if err == nil {
		err = os.WriteFile(fp, it.fileData, FilePerm)
	}

	return
}

// TODO fetch will be done from outside, and just populated.
func (it *Item) fetch(rp *Repo) {
	urlItem := utils.GetNewSuffixedUrl(rp.Url, it.fileName)

	it.fileData, it.netHttpCode, it.netFetchErr = rp.dumper.fetcher.Fetch(rp.dumper.app.Ctx, urlItem.String())
	it.fileSize = len(it.fileData)
	it.exists = it.netFetchErr == nil && it.netHttpCode != http.StatusNotFound
	it.fileDataStr = *(*string)(unsafe.Pointer(&it.fileData))
	// A string representation without copying the data. Will be overwritten if data are zlib compressed.
}

func (it *Item) GetReferences() (hashes []string, err error) {
	if !it.isObject {
		return it.findHashes(), nil
	}

	var data []byte
	data, err = utils.DecodeZlib(it.fileData)

	if err != nil {
		return
	}

	it.fileDataStr = string(data)
	start := it.fileDataStr[0:8]

	if strings.HasPrefix(start, "blob ") {
		return nil, nil
	} else if strings.HasPrefix(start, "tree ") {
		return it.parseGitTreeObject()
	} else if strings.HasPrefix(start, "commit ") {
		return it.findHashes(), nil
	} else if strings.HasPrefix(start, "tag ") {
		return nil, fmt.Errorf("unsupported git object type 'tag' in %s", it.fileName) // TODO
	}

	return
}

func (it *Item) findHashes() (hashes []string) {
	return regexp.MustCompile(HashRegexp).FindAllString(it.fileDataStr, application.LimitHashes)
}

func (it *Item) parseGitTreeObject() (hashes []string, err error) {
	reader := bytes.NewReader([]byte(it.fileDataStr))
	var mode []byte

	for {
		mode, err = readUntilByte(reader, ' ')

		if err != nil {
			if err == io.EOF {
				err = nil
			} else {
				err = fmt.Errorf("error reading 'mode' parameter: %v", err)
			}

			return
		}

		filename, err := readUntilByte(reader, 0)

		if err != nil {
			return hashes, fmt.Errorf("error reading 'filename' parameter: %v", err)
		}

		_ = mode
		_ = filename
		var hash [20]byte

		if _, err := io.ReadFull(reader, hash[:]); err != nil {
			return hashes, fmt.Errorf("error reading 'hash' parameter: %v", err)
		}

		hashes = append(hashes, hex.EncodeToString(hash[:]))
	}
}

func (it *Item) getRefFromHead() (reference string, isHash bool) {
	if strings.HasPrefix(it.fileDataStr, PrefixRef) {
		return strings.TrimSpace(strings.TrimPrefix(it.fileDataStr, PrefixRef)), false
	}

	// file contains just hash
	return strings.TrimSpace(it.fileDataStr), true
}
