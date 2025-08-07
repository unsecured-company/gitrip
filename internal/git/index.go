package git

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/go-git/go-git/v5/plumbing/format/index"
	"github.com/unsecured-company/gitrip/internal/application"
	"github.com/unsecured-company/gitrip/internal/utils"
)

type Index struct {
	Index *index.Index
}

func NewIndexFromFile(indexPath string) (idx *Index, err error) {
	file, err := os.Open(indexPath)

	if err != nil {
		return
	}

	idx, err = NewIndexFromReader(file)
	file.Close()

	return
}

func NewIndexFromReader(fr io.Reader) (idx *Index, err error) {
	idx = &Index{
		Index: &index.Index{},
	}

	err = index.NewDecoder(fr).Decode(idx.Index)

	return idx, err
}

func NewIndexFromBytes(data []byte) (idx *Index, err error) {
	fr := io.Reader(bytes.NewReader(data))

	return NewIndexFromReader(fr)
}

func RunIndexDump(app *application.App) (err error) {
	idx, err := NewIndexFromFile(app.Cfg.IndexFile)

	if err != nil {
		return fmt.Errorf("error reading index file: %w", err)
	}

	header := fmt.Sprintf("Index file '%s'", app.Cfg.IndexFile)
	var tree string

	if app.Cfg.Tree {
		header += " - TREE view"
		tree = utils.GetTreeAsString(idx.getFiles())
	} else if app.Cfg.Raw {
		header += " - RAW view"
		tree = idx.Index.String()
	} else if app.Cfg.Csv {
		header += " - CSV view"
		tree += idx.getAsCsv()
	} else {
		header += " - PATHS only"
		for _, ent := range idx.Index.Entries {
			tree += ent.Name + "\n"
		}
	}

	app.Out.Log(header)
	app.Out.Println(tree)

	return
}

func showAsTree(app *application.App) (err error) {
	idx, err := NewIndexFromFile(app.Cfg.IndexFile)

	if err != nil {
		return fmt.Errorf("error reading index file: %w", err)
	}

	header := fmt.Sprintf("Index file '%s'", app.Cfg.IndexFile)

	if app.Cfg.Raw {
		header += " - raw view"
	}

	if app.Cfg.Csv {
		header += " - only paths"
	}

	if app.Cfg.Tree {
		header += " - tree view"
	}

	app.Out.Println(header)

	var tree string

	if !app.Cfg.Tree && !app.Cfg.Raw && !app.Cfg.Csv {
		tree += idx.getAsCsv()
	}

	if app.Cfg.Tree {
		tree += idx.getAsTree()
		tree += utils.GetTreeAsString(idx.getFiles())
	}

	if app.Cfg.Raw {
		tree += idx.getAsRaw()
	}

	if app.Cfg.Csv {
		tree = idx.getAsPaths()
	}

	app.Out.Println(tree)

	return
}

func (idx *Index) getAsCsv() (str string) {
	str = "name;hash;size;created_at;modified_at\n"

	for _, e := range idx.Index.Entries {
		var modifiedAt string
		createdAt := e.CreatedAt.Format(time.DateTime)
		size := utils.SizeToHumanReadable(int64(e.Size))

		if e.CreatedAt != e.ModifiedAt {
			modifiedAt = e.ModifiedAt.Format(time.DateTime)
		}

		str += fmt.Sprintf("%s;%s;%s;%s;%s\n", e.Name, e.Hash.String(), size, createdAt, modifiedAt)
	}

	return
}

func (idx *Index) getAsTree() (str string) {
	for _, e := range idx.Index.Entries {
		size := utils.SizeToHumanReadable(int64(e.Size))
		str += size + "\t" + e.Name + "\n"
	}

	return
}

func (idx *Index) getAsPaths() (str string) {
	for _, ent := range idx.Index.Entries {
		str += ent.Name + "\n"
	}

	return
}

func (idx *Index) getAsRaw() (str string) {
	return idx.Index.String()
}

func (idx *Index) getFiles() (paths []string) {
	paths = make([]string, 0)

	for _, e := range idx.Index.Entries {
		paths = append(paths, e.Name)
	}

	return
}
