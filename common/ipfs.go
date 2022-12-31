package common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	shell "github.com/ipfs/go-ipfs-api"
	file "github.com/ipfs/go-ipfs-files"
	"os"
	"path/filepath"
	"strconv"
)

type IPFS struct {
	shell *shell.Shell
}

func NewIPFS(url string) *IPFS {
	return &IPFS{
		shell: shell.NewShell(url),
	}
}

func (i *IPFS) Upload(value string, index int64) (string, error) {
	tmppath, err := os.MkdirTemp("", strconv.FormatInt(index, 10))
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmppath)
	path := filepath.Join(tmppath, strconv.FormatInt(index, 10))
	b, err := json.Marshal(value)
	err = file.WriteTo(file.NewBytesFile(b), path)
	if err != nil {
		return "", err
	}
	hash, err := i.shell.AddDir(tmppath)
	if err != nil {
		fmt.Println("上传ipfs时错误：", err)
		return "", err
	}
	return hash, nil
}

func (i *IPFS) GenerateIPNS(cid string, index string) (*shell.Key, error) {
	return i.shell.KeyGen(context.Background(), fmt.Sprintf("%s-%s", cid, index), shell.KeyGen.Type("ed25519"))
}

func (i *IPFS) PublishWithDetails(cid string, name string) (string, error) {
	cidPath := fmt.Sprintf("/%s/%s", "ipfs", cid)
	resp, err := i.shell.PublishWithDetails(cidPath, name, 0, 0, false)
	if err != nil {
		return "", err
	}
	if resp.Value != cidPath {
		return "", errors.New(fmt.Sprintf("Expected to receive %s but got %s", cidPath, resp.Value))
	}
	return resp.Value, nil
}
