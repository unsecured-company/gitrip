package git

import (
	"bytes"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing/format/index"
	"io"
	"os"
	"unsecured.company/gitrip/internal/application"
	"unsecured.company/gitrip/internal/utils"
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

	if app.Cfg.Raw {
		header += " - raw view"
	}

	if app.Cfg.Tree {
		header += " - tree view"
	}

	app.Out.Println(header)

	var tree string

	if !app.Cfg.Tree && !app.Cfg.Raw {
		tree += idx.getAsTable()
	}

	if app.Cfg.Tree {
		tree += utils.GetTreeAsString(idx.getFiles())
	}

	if app.Cfg.Raw {
		tree += idx.Index.String()
	}

	app.Out.Println(tree)

	return
}

func (idx *Index) getAsTable() (str string) {
	for _, e := range idx.Index.Entries {
		size := utils.SizeToHumanReadable(int64(e.Size))
		str += size + "\t" + e.Name + "\n"
	}

	return
}

func (idx *Index) getFiles() (paths []string) {
	paths = make([]string, 0)

	for _, e := range idx.Index.Entries {
		paths = append(paths, e.Name)
	}

	return
}
