package ossweb

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type ossObjectNode struct {
	Root         string    `json:"root"`
	Path         string    `json:"path"`
	Type         string    `json:"type"`
	LastModified time.Time `json:"lastmodified"`
	FileSize     int64     `json:"filesize"`
	Filename     string    `json:"filename"`
}

type ossObjectNodes []*ossObjectNode

func (e ossObjectNodes) Len() int {
	return len(e)
}
func (e ossObjectNodes) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}
func (e ossObjectNodes) Less(i, j int) bool {
	if e[i].Type == "dir" && e[j].Type != "dir" {
		return true
	}
	if e[i].Type != "dir" && e[j].Type == "dir" {
		return false
	}
	return e[i].Path < e[j].Path
}

type structOssListCache struct {
	dir map[string][]*ossObjectNode
}

type ossClient struct {
	Endpoint        string
	AccessKeyId     string
	AccessKeySecret string
	Bucket          string
	Client          *oss.Client
	locker          sync.Mutex
	ossListCache    map[string]structOssListCache
	lastTime        time.Time
}

func NewOssClient(endpoint, accessKeyId, accessKeySecret, bucket string) (*ossClient, error) {

	client, err := oss.New(endpoint, accessKeyId, accessKeySecret)
	if err != nil {
		return nil, err
	}
	cli := &ossClient{
		Endpoint:        endpoint,
		AccessKeyId:     accessKeyId,
		AccessKeySecret: accessKeySecret,
		Bucket:          bucket,
		Client:          client,
	}
	cli.ossListCache = make(map[string]structOssListCache)
	cli.updateDir()
	return cli, nil
}

func (cli *ossClient) List(Root, Path string) ([]*ossObjectNode, error) {
	flagUpdateDir := false

	cli.locker.Lock()
	if _, ok := cli.ossListCache[Root]; !ok {
		flagUpdateDir = true
	}

	if cli.lastTime.Add(time.Second * 10).After(time.Now()) {
		flagUpdateDir = true
	}
	cli.locker.Unlock()

	if flagUpdateDir {
		go cli.updateDir()
	}
	if list, found := cli.ossListCache[Root]; found {
		if nodeList, foundDir := list.dir[Path]; foundDir {
			sort.Sort(ossObjectNodes(nodeList))
			return nodeList, nil
		}
	}
	return nil, nil
}

func (cli *ossClient) Put(Root, Path string, FileData io.Reader) error {
	bucket, err := cli.Client.Bucket(cli.Bucket)
	if err != nil {
		fmt.Println("bucket error:", err)
		return err
	}
	ossPath := filepath.ToSlash(filepath.Join(Root, Path))
	return bucket.PutObject(ossPath, FileData)
}

func (cli *ossClient) GetSignUrl(Root, Path string) (url string, err error) {
	bucket, err := cli.Client.Bucket(cli.Bucket)
	if err != nil {
		fmt.Println("bucket error:", err)
		return "", err
	}
	ossPath := filepath.ToSlash(filepath.Join(Root, Path))
	url, err = bucket.SignURL(ossPath, oss.HTTPGet, 60)
	return
}

func (cli *ossClient) Get(Root, Path string) (header http.Header, reader io.ReadCloser, err error) {
	bucket, err := cli.Client.Bucket(cli.Bucket)
	if err != nil {
		fmt.Println("bucket error:", err)
		return nil, nil, err
	}
	ossPath := filepath.ToSlash(filepath.Join(Root, Path))
	header, err = bucket.GetObjectMeta(ossPath)
	if err != nil {
		return nil, nil, err
	}
	reader, err = bucket.GetObject(ossPath)
	return
}

func (cli *ossClient) updateDir() {
	bucket, err := cli.Client.Bucket(cli.Bucket)
	if err != nil {
		fmt.Println("bucket error:", err)
		return
	}
	lsRes, err := bucket.ListObjectsV2()
	if err != nil {
		fmt.Println("ListObjectsV2 error:", err)
		return
	}
	ossListCache := make(map[string]structOssListCache)
	for _, object := range lsRes.Objects {
		fmt.Println("Objects:", object.Key)

		node := &ossObjectNode{}
		pathSplit := strings.Split(object.Key, "/")
		if len(pathSplit) < 2 {
			continue
		}

		node.Root = pathSplit[0]

		if _, found := ossListCache[node.Root]; !found {
			list := structOssListCache{}
			list.dir = make(map[string][]*ossObjectNode)
			ossListCache[node.Root] = list
		}

		node.Path = "/" + filepath.ToSlash(filepath.Join(pathSplit[1:]...))
		node.FileSize = object.Size
		node.LastModified = object.LastModified
		node.Filename = filepath.Base(node.Path)
		if pathSplit[len(pathSplit)-1] == "" {
			node.Type = "dir"
			// 如果是目录，则纳入上级目录的范畴
			_filepath := ""
			if len(pathSplit)-2 != 0 {
				_filepath = filepath.ToSlash(filepath.Join(pathSplit[1 : len(pathSplit)-2]...))
			}
			_filepath = "/" + _filepath
			ossListCache[node.Root].dir[_filepath] = append(ossListCache[node.Root].dir[_filepath], node)
		} else {
			node.Type = "file"
			//如果是文件,分割路径和文件名称
			_filepath := filepath.ToSlash(filepath.Join(pathSplit[1 : len(pathSplit)-1]...))
			_filepath = "/" + _filepath
			//fileName := pathSplit[len(pathSplit)-1]
			ossListCache[node.Root].dir[_filepath] = append(ossListCache[node.Root].dir[_filepath], node)
		}
	}

	cli.locker.Lock()
	cli.ossListCache = ossListCache
	cli.lastTime = time.Now()
	cli.locker.Unlock()

}
