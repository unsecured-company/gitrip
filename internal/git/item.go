package git

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
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

// TODO Better check for Exists

type Item struct {
	out           *application.Output
	doRefs        bool
	exists        bool
	isObject      bool
	isObjectValid bool
	fileData      []byte
	fileDataStr   string
	fileName      string
	fileSize      int
	netFetchErr   error
	netHttpCode   int
	fileDirPath   string
	regexpHash    *regexp.Regexp
	objectType    string
}

func NewItem(dirPath string, name string, doReferences bool, out *application.Output) (i *Item) {
	i = &Item{}
	i.out = out
	i.regexpHash = regexp.MustCompile(HashRegexp)
	i.fileDirPath = filepath.Join(dirPath, name)
	i.doRefs = doReferences
	i.fileName = name
	i.isObject = strings.HasPrefix(name, PathPrefixObjects)
	i.isObjectValid = i.isObject // by default valid

	return
}

func (it *Item) Save() (err error) {
	if it.exists == false {
		return
	}

	dir := filepath.Dir(it.fileDirPath)
	err = os.MkdirAll(dir, DirPerm)

	if err == nil {
		err = os.WriteFile(it.fileDirPath, it.fileData, FilePerm)
	}

	return
}

func (it *Item) Update(data []byte, code int, err error) {
	it.fileData = data
	it.netHttpCode = code
	it.netFetchErr = err
	it.fileSize = len(it.fileData)
	it.exists = it.netFetchErr == nil && it.netHttpCode != http.StatusNotFound
	it.fileDataStr = *(*string)(unsafe.Pointer(&it.fileData))
	// A string representation without copying the data. Will be overwritten if data are zlib compressed.
}

func (it *Item) GetPaths() (paths map[string]bool, err error) {
	if it.fileName == PathPrefixHooks {
		return
	}

	if it.fileName == PathPacked || it.fileName == PathInfoRefs {
		return it.getPathsFromPacked()
	}

	if it.fileName == PathPacks {
		return it.getPathsFromPacks()
	}

	if it.fileName == PathHead {
		return it.getRefFromHead()
	}

	if !it.isObject {
		return it.findHashes()
	}

	return it.getRefFromObject()
}

func (it *Item) IsValidIndexFile() bool {
	return len(it.fileData) >= 5 && strings.HasPrefix(string(it.fileData[:5]), PrefixDIRC)
}

func (it *Item) getPathFromIndexFile() (paths map[string]bool, err error) {
	paths = make(map[string]bool)
	fr := bytes.NewReader(it.fileData)
	index, err := NewIndexFromReader(fr)

	if err != nil {
		return paths, fmt.Errorf("Can not read index file: %v", err)
	}

	for _, e := range index.Index.Entries {
		path, _ := HashToPath(it.regexpHash, e.Hash.String())
		paths[path] = true
	}

	return
}

func (it *Item) getRefFromObject() (paths map[string]bool, err error) {
	paths = make(map[string]bool)
	var data []byte
	data, err = utils.DecodeZlib(it.fileData)

	if err != nil {
		return
	}

	it.isObjectValid, err = it.checkObjectData(data)

	if err != nil {
		if application.IgnoreInvalidObjectChecksum {
			it.out.Logf("%v | File will still be saved and analyzed.")
		} else {
			return
		}
	}

	it.fileDataStr = string(data)

	if len(it.fileDataStr) < 8 {
		return paths, fmt.Errorf("file too small: %s", it.fileName)
	}

	start := it.fileDataStr[0:8]

	if strings.HasPrefix(start, "blob ") {
		it.objectType = "blob"
		return
	} else if strings.HasPrefix(start, "tree ") {
		it.objectType = "tree"
		return it.parseGitTreeObject()
	} else if strings.HasPrefix(start, "commit ") {
		it.objectType = "commit"
		return it.findHashes()
	} else if strings.HasPrefix(start, "tag ") {
		it.objectType = "tag"
		return paths, fmt.Errorf("unsupported git object type 'tag' in %s", it.fileName) // TODO
	} else {
		it.objectType = "UNKNOWN"
		return paths, fmt.Errorf("unsupported git object type '%s' in %s", start, it.fileName)
	}
}

func (it *Item) checkObjectData(data []byte) (isValid bool, err error) {
	orig := strings.TrimPrefix(it.fileName, "objects/")
	orig = strings.ReplaceAll(orig, "/", "")

	if len(orig) != 40 {
		return false, fmt.Errorf("path is not object/XX/YYY..?")
	}

	h := sha1.New()
	h.Write(data)
	hash := hex.EncodeToString(h.Sum(nil))

	isValid = hash == orig

	if !isValid {
		err = fmt.Errorf("object [%s] sha1 has not valid hash [%s]", it.fileName, hash)
	}

	return
}

func (it *Item) CheckObject() (isValid bool, err error) {
	return
}

func (it *Item) findHashes() (paths map[string]bool, err error) {
	paths = make(map[string]bool)
	hashes := it.regexpHash.FindAllString(it.fileDataStr, application.LimitHashes)

	for _, hash := range hashes {
		path, errN := it.hashToPath(hash)
		paths[path] = true
		errors.Join(err, errN)
	}

	return
}

func (it *Item) parseGitTreeObject() (paths map[string]bool, err error) {
	paths = make(map[string]bool)
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
			return paths, fmt.Errorf("error reading 'filename' parameter: %v", err)
		}

		_ = mode
		_ = filename
		var hash [20]byte

		if _, err := io.ReadFull(reader, hash[:]); err != nil {
			return paths, fmt.Errorf("error reading 'hash' parameter: %v", err)
		}

		path, err := it.hashToPath(hex.EncodeToString(hash[:]))

		if err == nil {
			paths[path] = true
		}
	}
}

func (it *Item) getPathsFromPacked() (paths map[string]bool, err error) {
	paths = make(map[string]bool)
	lines := strings.Split(it.fileDataStr, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		parts := strings.Fields(line)

		if len(parts) != 2 {
			return paths, fmt.Errorf("Got %d instead of 2 parts for %s", len(parts), PathPacked)
		}

		pathH, err := it.hashToPath(parts[0])

		if err == nil {
			paths[pathH] = true
		} else {
			paths[parts[0]] = true
		}
		paths[parts[1]] = true
	}

	return
}

func (it *Item) getPathsFromPacks() (paths map[string]bool, err error) {
	paths = make(map[string]bool)
	hashes := it.regexpHash.FindAllString(it.fileDataStr, application.LimitHashes)

	for _, hash := range hashes {
		paths["objects/pack/pack-"+hash+".idx"] = true
		paths["objects/pack/pack-"+hash+".pack"] = true
		paths["objects/pack/pack-"+hash+".rev"] = true
	}

	return
}

func (it *Item) getRefFromHead() (paths map[string]bool, err error) {
	paths = make(map[string]bool)
	var path string

	if strings.HasPrefix(it.fileDataStr, PrefixRef) {
		path = strings.TrimSpace(strings.TrimPrefix(it.fileDataStr, PrefixRef))
		paths[path] = true
		paths["logs/"+path] = true
	} else {
		path, err = it.hashToPath(it.fileDataStr)

		if err == nil {
			paths[path] = true
		}
	}

	if err == nil && len(paths) == 0 {
		err = fmt.Errorf("no references found in %s: %s", PathHead, it.fileName)
	}

	return
}

func (it *Item) hashToPath(hash string) (path string, err error) {
	path, err = HashToPath(it.regexpHash, hash)

	if err != nil {
		err = fmt.Errorf("%w in %s", err, it.fileName)
	}

	return
}
