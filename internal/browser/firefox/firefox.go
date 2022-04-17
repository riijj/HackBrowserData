package firefox

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"

	"hack-browser-data/internal/browingdata"
	"hack-browser-data/internal/item"
	"hack-browser-data/internal/log"
	"hack-browser-data/internal/utils/fileutil"
	"hack-browser-data/internal/utils/typeutil"
)

type firefox struct {
	name        string
	storage     string
	profilePath string
	masterKey   []byte
	items       []item.Item
	itemPaths   map[item.Item]string
}

// New returns a new firefox instance.
func New(name, storage, profilePath string, items []item.Item) ([]*firefox, error) {
	if !fileutil.FolderExists(profilePath) {
		return nil, fmt.Errorf("%s profile folder is not exist: %s", name, profilePath)
	}
	f := &firefox{
		name:        name,
		storage:     storage,
		profilePath: profilePath,
		items:       items,
	}
	multiItemPaths, err := f.getMultiItemPath(f.profilePath, f.items)
	if err != nil {
		if strings.Contains(err.Error(), "profile folder is not exist") {
			log.Error(err)
			return nil, nil
		}
		return nil, err
	}
	var firefoxList []*firefox
	for name, itemPaths := range multiItemPaths {
		firefoxList = append(firefoxList, &firefox{
			name:      fmt.Sprintf("firefox-%s", name),
			items:     typeutil.Keys(itemPaths),
			itemPaths: itemPaths,
		})
	}
	return firefoxList, nil
}

func (f *firefox) getMultiItemPath(profilePath string, items []item.Item) (map[string]map[item.Item]string, error) {
	var multiItemPaths = make(map[string]map[item.Item]string)
	err := filepath.Walk(profilePath, firefoxWalkFunc(items, multiItemPaths))
	return multiItemPaths, err
}

func (f *firefox) copyItemToLocal() error {
	for i, path := range f.itemPaths {
		// var dstFilename = item.TempName()
		var filename = i.String()
		// TODO: Handle read file error
		d, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filename, d, 0777)
		if err != nil {
			return err
		}
	}
	return nil
}

func firefoxWalkFunc(items []item.Item, multiItemPaths map[string]map[item.Item]string) filepath.WalkFunc {
	return func(path string, info fs.FileInfo, err error) error {
		for _, v := range items {
			if info.Name() == v.FileName() {
				parentBaseDir := fileutil.ParentBaseDir(path)
				if _, exist := multiItemPaths[parentBaseDir]; exist {
					multiItemPaths[parentBaseDir][v] = path
				} else {
					multiItemPaths[parentBaseDir] = map[item.Item]string{v: path}
				}
			}
		}
		return err
	}
}

func (f *firefox) GetMasterKey() ([]byte, error) {
	return f.masterKey, nil
}

func (f *firefox) Name() string {
	return f.name
}

func (f *firefox) BrowsingData() (*browingdata.Data, error) {
	b := browingdata.New(f.items)

	if err := f.copyItemToLocal(); err != nil {
		return nil, err
	}

	masterKey, err := f.GetMasterKey()
	if err != nil {
		return nil, err
	}

	f.masterKey = masterKey
	if err := b.Recovery(f.masterKey); err != nil {
		return nil, err
	}
	return b, nil
}
