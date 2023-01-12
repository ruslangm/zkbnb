package common

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	shell "github.com/ipfs/go-ipfs-api"
	file "github.com/ipfs/go-ipfs-files"
	"github.com/mr-tron/base58/base58"
	"github.com/zeromicro/go-zero/core/logx"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type IPFS struct {
	shell *shell.Shell
}

var Ipfs *IPFS

func NewIPFS(url string) *IPFS {
	Ipfs = &IPFS{
		shell: shell.NewShell(url),
	}
	return Ipfs
}

func (i *IPFS) Upload(value []byte, index int64) (string, error) {
	tmppath, err := os.MkdirTemp("", strconv.FormatInt(index, 10))
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmppath)
	path := filepath.Join(tmppath, strconv.FormatInt(index, 10))
	err = file.WriteTo(file.NewBytesFile(value), path)
	if err != nil {
		return "", err
	}
	hash, err := i.shell.AddDir(tmppath)
	if err != nil {
		fmt.Println("上传ipfs时错误：", err)
		return "", err
	}
	return hash, err
}

func (i *IPFS) UploadIPNS(value string) (string, error) {
	cid, err := i.shell.Add(bytes.NewBufferString(value))
	if err != nil {
		return "", err
	}
	return cid, err
}

func (i *IPFS) GenerateIPNS(ipnsName string) (*shell.Key, error) {
	return i.shell.KeyGen(context.Background(), ipnsName, shell.KeyGen.Type("ed25519"))
}

func (i *IPFS) PublishWithDetails(cid string, name string) (string, error) {
	cidPath := fmt.Sprintf("/%s/%s", "ipfs", cid)
	resp, err := i.shell.PublishWithDetails(cidPath, name, 0, 0, false)
	if err != nil {
		return "", err
	}
	if resp.Value != cidPath {
		logx.Severe(fmt.Sprintf("Expected to receive %s but got %s", cidPath, resp.Value))
		return "", errors.New(fmt.Sprintf("Expected to receive %s but got %s", cidPath, resp.Value))
	}
	return resp.Value, nil
}

func (i *IPFS) GenerateHash(cid string) (string, error) {
	base, err := base58.Decode(cid)
	if err != nil {
		return "", err
	}
	hex := hexutil.Encode(base)
	lowerHex := strings.ToLower(hex)
	return strings.Replace(lowerHex, "0x1220", "", 1), nil
}

func (i *IPFS) GenerateCid() (nftContentHash string) {
	var hash = "0x1220" + nftContentHash
	b, _ := hexutil.Decode(hash)
	cid := base58.Encode(b)
	return cid
}
