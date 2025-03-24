package git

import (
	"bytes"
	"compress/zlib"
	"github.com/stretchr/testify/assert"
	"testing"
)

type refTest struct {
	name    string
	content string
	refs    []string
}

func TestGetReferencesFromRegularFile(t *testing.T) {
	refsDomina := refTest{
		name:    "/refs/heads/domina",
		content: "0000000000000000000000000000000000000000 2b9c3f3aae0c83775239dc2b04301d833382a497 Unsecured Company <git@unsecured.company> 1742629735 +0100\tcommit (initial): Batman\n",
		refs:    []string{"0000000000000000000000000000000000000000", "2b9c3f3aae0c83775239dc2b04301d833382a497"},
	}

	refsDominaItem := createItem(refsDomina.name, refsDomina.content)
	refs, err := refsDominaItem.GetReferences()

	assert.NoError(t, err)
	assert.ElementsMatch(t, refs, refsDomina.refs)
	t.Log(err)
	t.Log(refs)
}

func createItem(fileName string, content string) *Item {
	return &Item{
		fileName:    fileName,
		fileData:    CompressGitObject([]byte(content)),
		fileDataStr: content,
	}
}

func CompressGitObject(data []byte) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}
