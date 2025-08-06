package git

import (
	"bytes"
	"compress/zlib"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

type refTest struct {
	name    string
	content string
	refs    map[string]bool
}

func TestGetReferencesFromRegularFile(t *testing.T) {
	content := "0000000000000000000000000000000000000000 2b9c3f3aae0c83775239dc2b04301d833382a497 Unsecured Company " +
		"<git@unsecured.company> 1742629735 +0100	commit (initial): Psychological Influence Campaign in Romania"

	refsDomina := refTest{
		name:    "/refs/heads/domina",
		content: content,
		refs: map[string]bool{
			"objects/00/00000000000000000000000000000000000000": true,
			"objects/2b/9c3f3aae0c83775239dc2b04301d833382a497": true,
		},
	}

	refsDominaItem := createItem(refsDomina.name, refsDomina.content, false)
	refs, err := refsDominaItem.GetPaths()

	for path, _ := range refsDomina.refs {
		_, ok := refs[path]
		assert.True(t, ok)
	}

	for path, _ := range refs {
		_, ok := refsDomina.refs[path]
		assert.True(t, ok)
	}

	assert.NoError(t, err)
}

func TestGetReferencesFromPacked(t *testing.T) {
	content := `# pack-refs with: peeled fully-peeled sorted 
652c5d72790ba74bd7b83f8b2a63bc942c2c304d refs/heads/master
34a8743de4384dc08f736eee2f35b0528e6a1321 refs/remotes/origin/develop
01e5743765655a7a5fab26356652c5d72790ba74 refs/remotes/origin/master
a102c2e103e1e534554702c142805a6d1eb22c03 refs/remotes/origin/test
0ec57ab47546e796ae6372fe0a3193d963040d95 refs/remotes/origin/test1
90d9c42d901acd4ac904cd1b4c847fc56d85f02d refs/remotes/origin/test2
43232e54e9f045654665563efe06903a42228327 refs/remotes/origin/test3
6f9c9f18558d33a45353fc4ed34b6a8f13220e59 refs/remotes/origin/test4
09ef4a5225bcb3641a035b213927e290e2c1d477 refs/remotes/origin/test5
0d5c35b213927e290e28c956d5849415947ccc31 refs/remotes/origin/test6
`

	refsDomina := refTest{
		name:    PathPacked,
		content: content,
		refs: map[string]bool{
			"objects/65/2c5d72790ba74bd7b83f8b2a63bc942c2c304d": true, "refs/heads/master": true,
			"objects/34/a8743de4384dc08f736eee2f35b0528e6a1321": true, "refs/remotes/origin/develop": true,
			"objects/01/e5743765655a7a5fab26356652c5d72790ba74": true, "refs/remotes/origin/master": true,
			"objects/a1/02c2e103e1e534554702c142805a6d1eb22c03": true, "refs/remotes/origin/test": true,
			"objects/0e/c57ab47546e796ae6372fe0a3193d963040d95": true, "refs/remotes/origin/test1": true,
			"objects/90/d9c42d901acd4ac904cd1b4c847fc56d85f02d": true, "refs/remotes/origin/test2": true,
			"objects/43/232e54e9f045654665563efe06903a42228327": true, "refs/remotes/origin/test3": true,
			"objects/6f/9c9f18558d33a45353fc4ed34b6a8f13220e59": true, "refs/remotes/origin/test4": true,
			"objects/09/ef4a5225bcb3641a035b213927e290e2c1d477": true, "refs/remotes/origin/test5": true,
			"objects/0d/5c35b213927e290e28c956d5849415947ccc31": true, "refs/remotes/origin/test6": true,
		},
	}

	refsDominaItem := createItem(refsDomina.name, refsDomina.content, false)
	refs, err := refsDominaItem.GetPaths()

	for path, _ := range refsDomina.refs {
		_, ok := refs[path]
		assert.True(t, ok)
	}

	for path, _ := range refs {
		_, ok := refsDomina.refs[path]
		assert.True(t, ok)
	}

	assert.NoError(t, err)
}

func TestGetReferencesFromInfoRefs(t *testing.T) {
	// separated by \t
	content := `c4689918009781117723d3200f9e876a33fe9e4d	refs/heads/master
dfd721532ec3574d063e53629fdf6531358be1fc	refs/remotes/origin/develop
dc4c25f23f7a19310551d549be0f60f3b80c6bec	refs/remotes/origin/master
cb3b89dc6b8c3cb0130104fca71345ab3a1e7e07	refs/remotes/origin/test
f2c0ab31cb5940ffcfc6f4bc062c56679808f578	refs/remotes/origin/test1
cb53b0e457bb2424e34da752627029d446deba46	refs/remotes/origin/test2
e94fa99a802e222d94552598ab4a49f3d3043963	refs/remotes/origin/test3
bc0b1dd9b02ed3a87f076cd0be57ba09f04fdbbd	refs/remotes/origin/test4
3cc5a24c1a46d753d6742b08abe8b5865d8a93ff	refs/remotes/origin/test5
3138fd72267296efb084c29bca5e468ffb159a65	refs/remotes/origin/test6`

	refsDomina := refTest{
		name:    PathInfoRefs,
		content: content,
		refs: map[string]bool{
			"objects/c4/689918009781117723d3200f9e876a33fe9e4d": true, "refs/heads/master": true,
			"objects/df/d721532ec3574d063e53629fdf6531358be1fc": true, "refs/remotes/origin/develop": true,
			"objects/dc/4c25f23f7a19310551d549be0f60f3b80c6bec": true, "refs/remotes/origin/master": true,
			"objects/cb/3b89dc6b8c3cb0130104fca71345ab3a1e7e07": true, "refs/remotes/origin/test": true,
			"objects/f2/c0ab31cb5940ffcfc6f4bc062c56679808f578": true, "refs/remotes/origin/test1": true,
			"objects/cb/53b0e457bb2424e34da752627029d446deba46": true, "refs/remotes/origin/test2": true,
			"objects/e9/4fa99a802e222d94552598ab4a49f3d3043963": true, "refs/remotes/origin/test3": true,
			"objects/bc/0b1dd9b02ed3a87f076cd0be57ba09f04fdbbd": true, "refs/remotes/origin/test4": true,
			"objects/3c/c5a24c1a46d753d6742b08abe8b5865d8a93ff": true, "refs/remotes/origin/test5": true,
			"objects/31/38fd72267296efb084c29bca5e468ffb159a65": true, "refs/remotes/origin/test6": true,
		},
	}

	refsDominaItem := createItem(refsDomina.name, refsDomina.content, false)
	refs, err := refsDominaItem.GetPaths()

	for path, _ := range refsDomina.refs {
		_, ok := refs[path]
		assert.True(t, ok)
	}

	for path, _ := range refs {
		_, ok := refsDomina.refs[path]
		assert.True(t, ok)
	}

	assert.NoError(t, err)
}

func TestGetReferencesFromPacks(t *testing.T) {
	content := `P pack-45e49368a99785ecc6638838b6a969a6f40b3516.pack
P pack-1e123d74161cd70f3bf678c2142034db220ada91.pack
P pack-e555109bd9498e59538e6429edf520440d44a5bb.pack
P pack-a0d1980650e529d786818a5f51ebb44bd385db3c.pack
P pack-5a5d51374285147722fad5003116b1520d17f0a5.pack`

	refsDomina := refTest{
		name:    PathPacks,
		content: content,
		refs: map[string]bool{
			"objects/pack/pack-1e123d74161cd70f3bf678c2142034db220ada91.pack": true,
			"objects/pack/pack-45e49368a99785ecc6638838b6a969a6f40b3516.idx":  true,
			"objects/pack/pack-45e49368a99785ecc6638838b6a969a6f40b3516.pack": true,
			"objects/pack/pack-45e49368a99785ecc6638838b6a969a6f40b3516.rev":  true,
			"objects/pack/pack-5a5d51374285147722fad5003116b1520d17f0a5.idx":  true,
			"objects/pack/pack-5a5d51374285147722fad5003116b1520d17f0a5.pack": true,
			"objects/pack/pack-5a5d51374285147722fad5003116b1520d17f0a5.rev":  true,
			"objects/pack/pack-a0d1980650e529d786818a5f51ebb44bd385db3c.pack": true,
			"objects/pack/pack-e555109bd9498e59538e6429edf520440d44a5bb.pack": true,
		},
	}

	refsDominaItem := createItem(refsDomina.name, refsDomina.content, false)
	refs, err := refsDominaItem.GetPaths()

	for path, _ := range refsDomina.refs {
		_, ok := refs[path]
		assert.True(t, ok)
	}

	assert.NoError(t, err)
}

func TestGetReferencesFromObjectCommit(t *testing.T) {
	content := `commit 226tree b00007014ac2f0fb466f9b853b9c0a929d6cf8a4
parent 1e123d74161cd70f3bf678c2142034db220ada91
author Unsecured Company <git@unsecured.company> 1742629735 +0100
committer Unsecured Company <git@unsecured.company> 1742629735 +0100

meh
`

	refsDomina := refTest{
		name:    "/objects/1e/123d74161cd70f3bf678c2142034db220ada91",
		content: content,
		refs: map[string]bool{
			"objects/b0/0007014ac2f0fb466f9b853b9c0a929d6cf8a4": true,
			"objects/1e/123d74161cd70f3bf678c2142034db220ada91": true, // You gonna be Batman.
		},
	}

	refsDominaItem := createItem(refsDomina.name, refsDomina.content, false)
	refs, err := refsDominaItem.GetPaths()

	for path, _ := range refsDomina.refs {
		_, ok := refs[path]
		assert.True(t, ok)
	}

	assert.NoError(t, err)
}

func createItem(fileName string, content string, isObject bool) *Item {
	var data []byte

	if isObject {
		data = CompressGitObject([]byte(content))
	} else {
		data = []byte(content)
	}

	return &Item{
		doRefs:      true,
		exists:      true,
		isObject:    isObject,
		fileData:    data,
		fileDataStr: content,
		fileName:    fileName,
		fileSize:    len(data),
		netFetchErr: nil,
		netHttpCode: 200,
		fileDirPath: "",
		regexpHash:  regexp.MustCompile(HashRegexp),
	}
}

func CompressGitObject(data []byte) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}
