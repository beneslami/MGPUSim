// Code generated by "esc -pkg fir --private -o esc.go kernels.hsaco"; DO NOT EDIT.

package fir

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync"
	"time"
)

type _escLocalFS struct{}

var _escLocal _escLocalFS

type _escStaticFS struct{}

var _escStatic _escStaticFS

type _escDirectory struct {
	fs   http.FileSystem
	name string
}

type _escFile struct {
	compressed string
	size       int64
	modtime    int64
	local      string
	isDir      bool

	once sync.Once
	data []byte
	name string
}

func (_escLocalFS) Open(name string) (http.File, error) {
	f, present := _escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	return os.Open(f.local)
}

func (_escStaticFS) prepare(name string) (*_escFile, error) {
	f, present := _escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	var err error
	f.once.Do(func() {
		f.name = path.Base(name)
		if f.size == 0 {
			return
		}
		var gr *gzip.Reader
		b64 := base64.NewDecoder(base64.StdEncoding, bytes.NewBufferString(f.compressed))
		gr, err = gzip.NewReader(b64)
		if err != nil {
			return
		}
		f.data, err = ioutil.ReadAll(gr)
	})
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (fs _escStaticFS) Open(name string) (http.File, error) {
	f, err := fs.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.File()
}

func (dir _escDirectory) Open(name string) (http.File, error) {
	return dir.fs.Open(dir.name + name)
}

func (f *_escFile) File() (http.File, error) {
	type httpFile struct {
		*bytes.Reader
		*_escFile
	}
	return &httpFile{
		Reader:   bytes.NewReader(f.data),
		_escFile: f,
	}, nil
}

func (f *_escFile) Close() error {
	return nil
}

func (f *_escFile) Readdir(count int) ([]os.FileInfo, error) {
	if !f.isDir {
		return nil, fmt.Errorf(" escFile.Readdir: '%s' is not directory", f.name)
	}

	fis, ok := _escDirs[f.local]
	if !ok {
		return nil, fmt.Errorf(" escFile.Readdir: '%s' is directory, but we have no info about content of this dir, local=%s", f.name, f.local)
	}
	limit := count
	if count <= 0 || limit > len(fis) {
		limit = len(fis)
	}

	if len(fis) == 0 && count > 0 {
		return nil, io.EOF
	}

	return fis[0:limit], nil
}

func (f *_escFile) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *_escFile) Name() string {
	return f.name
}

func (f *_escFile) Size() int64 {
	return f.size
}

func (f *_escFile) Mode() os.FileMode {
	return 0
}

func (f *_escFile) ModTime() time.Time {
	return time.Unix(f.modtime, 0)
}

func (f *_escFile) IsDir() bool {
	return f.isDir
}

func (f *_escFile) Sys() interface{} {
	return f
}

// _escFS returns a http.Filesystem for the embedded assets. If useLocal is true,
// the filesystem's contents are instead used.
func _escFS(useLocal bool) http.FileSystem {
	if useLocal {
		return _escLocal
	}
	return _escStatic
}

// _escDir returns a http.Filesystem for the embedded assets on a given prefix dir.
// If useLocal is true, the filesystem's contents are instead used.
func _escDir(useLocal bool, name string) http.FileSystem {
	if useLocal {
		return _escDirectory{fs: _escLocal, name: name}
	}
	return _escDirectory{fs: _escStatic, name: name}
}

// _escFSByte returns the named file from the embedded assets. If useLocal is
// true, the filesystem's contents are instead used.
func _escFSByte(useLocal bool, name string) ([]byte, error) {
	if useLocal {
		f, err := _escLocal.Open(name)
		if err != nil {
			return nil, err
		}
		b, err := ioutil.ReadAll(f)
		_ = f.Close()
		return b, err
	}
	f, err := _escStatic.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.data, nil
}

// _escFSMustByte is the same as _escFSByte, but panics if name is not present.
func _escFSMustByte(useLocal bool, name string) []byte {
	b, err := _escFSByte(useLocal, name)
	if err != nil {
		panic(err)
	}
	return b
}

// _escFSString is the string version of _escFSByte.
func _escFSString(useLocal bool, name string) (string, error) {
	b, err := _escFSByte(useLocal, name)
	return string(b), err
}

// _escFSMustString is the string version of _escFSMustByte.
func _escFSMustString(useLocal bool, name string) string {
	return string(_escFSMustByte(useLocal, name))
}

var _escData = map[string]*_escFile{

	"/kernels.hsaco": {
		name:    "kernels.hsaco",
		local:   "kernels.hsaco",
		size:    9536,
		modtime: 1593104816,
		compressed: `
H4sIAAAAAAAC/+xa22/bVBj/fHLFvdANCXaRwKBJHdPiOu5laV/ojZRobde17AaahmufpGa+RI5dNUhk
abm9TGLiiT+Cv2EJb3vmeQ88sP0HIJ4Isn1OYnv11omhIeafFP98vu98t+PjyMc+dz5cLSOGmQeCFPwK
jHsy5repYv49n895shLkYR6GgYUsAKSD/SLcZcKcJ3KG2MVheiTMNB/XLhNoR/k3JsxBOzdX4Ig8wnUI
M7VDz2lH69t8ZCvpI9gF83Nx+ZGtZOH5kabjiWCQeIAfsGFOB+zyJP7C2rLXnV6b09588OVpyPVro7KF
teWVjSt+35MAMETkkq7UZKMg6Yr722lIhUKtulcSJonfMgvAkr6FQoG9iq2GahpzHMWnXPE8J3A32YvY
MrDWmGM5rsCtSzoe9OE4rlzZZF3eaurbphZQj5crm/O3lXFPuyoZNUeqDUwv1bGxtMothbT9HLzYInfT
0y5YNS+4i0MSMB277tgsbX7crONQn/GqZkr2ufF+jy31i7CHUl+1oKk1Y+5Q1VVJc/BF1VCoekUztyVt
0alWsRXu5aZAe5UnxYF3RbG26pKMLzuSNtd3MdDLMtX4WMZVydHs+NplE1err2bpqvHKXvUdtWGbVvPV
LN5w9Fu2VI8v3lENO77wqfjCp+ILX2x6oviarwRrflZNL+hqfKQqCjb8Ab1UrTawfT0+wcrM1L8f/8ZL
jv/JS4i/bhpPmReV0hFvhSSrl5bVmqPZ6oqlKltNQ16wav84wyVTwRuWWe8/tbhPUJJV28I1HRu2n/zs
DFGuWKZTJ6qyuocVXy8Q9Yal7ko2ju8Qdk7Kp4lek3Zx1TJpUI7r3wbrjr61srHZ6A+VKA40V0OaIg21
Ju2VNcm+Zlq3/aw9p+L0DMvzPBv/XOw+y55KZZ9YHzCB36nAGsZt1x5/N4TIozMTdViubEKCBP8TMP15
nvdXdszT+x/AT3DPW+uFb7rPAudvRnTpdDrb6/V6/8X6U6nRrr+mRV13af4VZLtnAWARUNf9T2ijdOsO
3O0wPeZbt4I8k/8hDyAiABHa2U4b9u+/Dd902ijfGmUO7kMKtVwbBn78ElC2dQyxrTfQcGsMjbaOo7FW
O5uFzMhxyB0bhVQqK2bg+18Osgh+RkjMDbHiayPDJYCNhxmAFMMgLzdmn9lHX6N2qp06ACbXqcPdTjqb
Pv+458ZFLXB9AIIRlG0NA4ipFCoB1B/6a//9TjLLEyRIkCBBggQJEiRIkCAB/db8YNjnIdI+QThDWCDf
4dmI/Pe/eqbLfxJ7+l35xMjh8VZV4za25rjV1WWuWOQFXuDOTjQseQLv2dgyJG1C03b1Qt0yP8eyPaFp
CjcjC6XtWUGYnp3FGE+LuChuy7iIL8xsi4IiTxUnZTwp4Avvg6xJRo3b9T/lHsW9b3D0AE9/j+KO5vVj
YXmOyLsR+Qgd/bGw/LR7QLnBfgGCt2LeYwJvmDYGXmkajaYOfM1w+B2psQPk6MptC3gb79leS9JVGXjZ
1HVs2MA3mrotbQPf2GnYln/mMywuCrdE7zgJ5crmGc2UJc17/Xlr+cb6wlpl6UW9f8oFtivE7Vvov0uC
J8d3KGBG5zFlITBfmcD+DDq/XweAP3o9k9rTeUyZi6SVj8Q/SXyjyLzvc8Q+HeF3yH4KFLnPKGcOnWcD
jAf3tkD8fpg4BwVim6KCmH0qmUj9NMwMcSlEwtSJfScmPOUPgtc+AOFdn+/B4H8pc8j1WwnmHkCX2N94
xvhdjrE/SfY7nXmG/d8BAAD//zqf1cJAJQAA
`,
	},
}

var _escDirs = map[string][]os.FileInfo{}
