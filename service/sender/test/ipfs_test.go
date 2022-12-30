package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	shell "github.com/ipfs/go-ipfs-api"
	file "github.com/ipfs/go-ipfs-files"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// 交易结构体(未来的通道)
type Transaction struct {
	Person1      string `json:"person1,omitempty" xml:"person1"`
	Person2      string `json:"person2,omitempty" xml:"person2"`
	Person1money string `json:"person1Money,omitempty" xml:"person1Money"`
	Person2money string `json:"person2Money,omitempty" xml:"person2Money"`
}

// 数据上传到ipfs
func UploadIPFS(shell *shell.Shell, str string) string {
	hash, err := shell.Add(bytes.NewBufferString(str))
	if err != nil {
		fmt.Println("上传ipfs时错误：", err)
	}
	return hash
}

// 数据上传到ipfs
func UploadFile(shell *shell.Shell, str []byte, token int64) string {
	tmppath, err := os.MkdirTemp("/Users/user", "files-test")
	if err != nil {
		return ""
	}
	defer os.RemoveAll(tmppath)
	path := filepath.Join(tmppath, strconv.FormatInt(token, 10))
	err = file.WriteTo(file.NewBytesFile(str), path)
	if err != nil {
		return ""
	}
	hash, err := shell.AddDir(tmppath)
	if err != nil {
		fmt.Println("上传ipfs时错误：", err)
	}
	return hash
}

// 从ipfs下载数据
func CatIPFS(shell *shell.Shell, hash string) string {

	read, err := shell.Cat(hash)
	if err != nil {
		fmt.Println(err)
	}
	body, err := ioutil.ReadAll(read)

	return string(body)
}

// 通道序列化
func marshalStruct(transaction Transaction) []byte {

	data, err := json.Marshal(&transaction)
	if err != nil {
		fmt.Println("序列化err=", err)
	}
	return data
}

// 数据反序列化为通道
func unmarshalStruct(str []byte) Transaction {
	var transaction Transaction
	err := json.Unmarshal(str, &transaction)
	if err != nil {
		fmt.Println("unmarshal err:", err)
	}
	return transaction
}

func TestGet1Txs(t *testing.T) {
	sh := shell.NewShell("localhost:5001")
	//生成一个交易结构体(未来的通道)
	transaction := Transaction{
		Person1:      "Aaron1211",
		Person2:      "Bob11",
		Person1money: "1000",
		Person2money: "20011",
	}
	//结构体序列化
	data := marshalStruct(transaction)

	//上传到ipfs
	hash1 := UploadFile(sh, data, 1)
	fmt.Println("文件hash1是", hash1)

	//上传到ipfs
	hash := UploadIPFS(sh, string(data))
	fmt.Println("文件hash是", hash)
	//从ipfs下载数据
	str2 := CatIPFS(sh, hash)
	//数据反序列化
	transaction2 := unmarshalStruct([]byte(str2))
	//验证下数据
	fmt.Println(transaction2)

}

func TestKeyIpns(t *testing.T) {
	sh := shell.NewShell("localhost:5001")
	key1, _ := sh.KeyGen(context.Background(), "cid+index", shell.KeyGen.Type("rsa"))
	fmt.Println(key1)
}

func TestPublish(t *testing.T) {
	sh := shell.NewShell("localhost:5001")
	resp, err := sh.PublishWithDetails("/ipfs/"+"hash", "cid+index", 0, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(resp.Value)
}
