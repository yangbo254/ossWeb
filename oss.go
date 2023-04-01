package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type ossObjectNode struct {
	Root         string
	Path         string
	Type         string
	LastModified time.Time
	FileSize     int64
}

type structOssListCache struct {
	dir map[string][]ossObjectNode
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

func (cli *ossClient) List(Root, Path string) ([]ossObjectNode, error) {
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
	ossPath := filepath.ToSlash(filepath.Join("/"+Root, Path))
	return bucket.PutObject(ossPath, FileData)
}

func (cli *ossClient) Get(Root, Path string) (io.ReadCloser, error) {
	bucket, err := cli.Client.Bucket(cli.Bucket)
	if err != nil {
		fmt.Println("bucket error:", err)
		return nil, err
	}
	ossPath := filepath.ToSlash(filepath.Join("/"+Root, Path))
	return bucket.GetObject(ossPath)
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

		node := ossObjectNode{}
		pathSplit := strings.Split(object.Key, "/")
		if len(pathSplit) < 2 {
			continue
		}

		node.Root = pathSplit[0]

		if _, found := ossListCache[node.Root]; !found {
			list := structOssListCache{}
			list.dir = make(map[string][]ossObjectNode)
			ossListCache[node.Root] = list
		}

		node.Path = filepath.ToSlash(filepath.Join(pathSplit[1:]...))
		node.FileSize = object.Size
		node.LastModified = object.LastModified

		if pathSplit[len(pathSplit)-1] == "" {
			node.Type = "dir"
			// 如果是目录，则纳入上级目录的范畴
			filePath := filepath.ToSlash(filepath.Join(pathSplit[1 : len(pathSplit)-2]...))
			filePath = "/" + filePath
			ossListCache[node.Root].dir[filePath] = append(ossListCache[node.Root].dir[filePath], node)
		} else {
			node.Type = "file"
			//如果是文件,分割路径和文件名称
			filePath := filepath.ToSlash(filepath.Join(pathSplit[1 : len(pathSplit)-1]...))
			filePath = "/" + filePath
			//fileName := pathSplit[len(pathSplit)-1]
			ossListCache[node.Root].dir[filePath] = append(ossListCache[node.Root].dir[filePath], node)
		}
	}

	cli.locker.Lock()
	cli.ossListCache = ossListCache
	cli.lastTime = time.Now()
	cli.locker.Unlock()

}
