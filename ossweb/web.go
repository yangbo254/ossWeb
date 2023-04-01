package ossweb

import (
	b64 "encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type webEngine struct {
	oss *ossClient
}

type defaultAck struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func NewWebEngine() (*webEngine, error) {
	web := &webEngine{}
	ossEndpoint, err := GetConfigValue("ossEndpoint")
	if err != nil {
		return nil, errors.New("not found oss endpoint info in config")
	}
	ossAccessKeyId, err := GetConfigValue("ossAccessKeyId")
	if err != nil {
		return nil, errors.New("not found oss accesskey id info in config")
	}
	ossAccessKeySecret, err := GetConfigValue("ossAccessKeySecret")
	if err != nil {
		return nil, errors.New("not found oss accesskey secret info in config")
	}
	ossBucket, err := GetConfigValue("ossBucket")
	if err != nil {
		return nil, errors.New("not found oss bucket info in config")
	}

	ossClient, err := NewOssClient(ossEndpoint.(string), ossAccessKeyId.(string), ossAccessKeySecret.(string), ossBucket.(string))
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	web.oss = ossClient
	return web, nil
}

func (web *webEngine) Run() error {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	//	@BasePath	/api

	// ListDir godoc
	//	@Summary	List the directory tree as root and path
	//	@Schemes
	//	@Description	List the directory tree as root and path
	//	@Tags			example
	//	@Accept			json
	//	@Produce		json
	//	@Success		200	{defaultAck}	code==0
	//	@Router			/api/list [post]
	r.POST("/api/list", func(ctx *gin.Context) {
		type listReq struct {
			Path string `json:"path"`
		}
		var req listReq

		auth, err := web.tokenAuth(ctx)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, defaultAck{
				Code:    1,
				Message: err.Error(),
			})
			return
		}

		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, defaultAck{
				Code:    2,
				Message: "invalid json: " + err.Error(),
			})
			return
		}

		list, err := web.oss.List(auth.Username, req.Path)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, defaultAck{
				Code:    3,
				Message: err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusOK, defaultAck{
			Code:    0,
			Message: "success",
			Data:    list,
		})

	})

	r.POST("/api/get", func(ctx *gin.Context) {
		type listReq struct {
			Path string `json:"path"`
		}
		var req listReq

		auth, err := web.tokenAuth(ctx)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, defaultAck{
				Code:    1,
				Message: err.Error(),
			})
			return
		}

		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, defaultAck{
				Code:    2,
				Message: "invalid json: " + err.Error(),
			})
			return
		}

		data, err := web.oss.Get(auth.Username, req.Path)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, defaultAck{
				Code:    3,
				Message: err.Error(),
			})
			return
		}
		var byteData []byte
		_, err = data.Read(byteData)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, defaultAck{
				Code:    3,
				Message: err.Error(),
			})
			return
		}

		ctx.JSON(http.StatusOK, defaultAck{
			Code:    0,
			Message: "success",
			Data:    b64.StdEncoding.EncodeToString(byteData),
		})

	})

	r.POST("/api/put", func(ctx *gin.Context) {
		auth, err := web.tokenAuth(ctx)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, defaultAck{
				Code:    1,
				Message: err.Error(),
			})
			return
		}

		file, err := ctx.FormFile("file")
		if err != nil {
			ctx.JSON(http.StatusBadRequest, defaultAck{
				Code:    2,
				Message: fmt.Sprintf("get data error %v", err),
			})
			return
		}
		if file.Filename == "" {
			ctx.JSON(http.StatusBadRequest, defaultAck{
				Code:    2,
				Message: err.Error(),
			})
			return
		}
		paths := strings.Split(filepath.ToSlash(file.Filename), "/")
		if paths[len(paths)-1] == "" {
			ctx.JSON(http.StatusBadRequest, defaultAck{
				Code:    2,
				Message: err.Error(),
			})
			return
		}
		src, err := file.Open()
		if err != nil {
			ctx.JSON(http.StatusBadRequest, defaultAck{
				Code:    2,
				Message: err.Error(),
			})
			return
		}
		defer src.Close()
		err = web.oss.Put(auth.Username, filepath.ToSlash(file.Filename), src)
		if err != nil {
			ctx.JSON(http.StatusUnprocessableEntity, defaultAck{
				Code:    3,
				Message: err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusUnprocessableEntity, defaultAck{
			Code:    0,
			Message: "success",
		})

	})

	apiPort, err := GetConfigValue("apiPort")
	if err != nil {
		return errors.New("not found apii port info in config")
	}
	r.Run(fmt.Sprintf("0.0.0.0:%v", apiPort.(int))) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
	return nil
}

func (web *webEngine) tokenAuth(c *gin.Context) (*Auth, error) {
	var authorization string
	bearToken := c.Request.Header.Get("Authorization")
	strArr := strings.Split(bearToken, " ")
	if len(strArr) == 2 && strArr[0] == "Bearer" {
		authorization = strArr[1]
	}
	auth := &Auth{}
	err := auth.CheckToken(authorization)
	return auth, err
}
