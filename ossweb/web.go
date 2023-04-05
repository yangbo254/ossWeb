package ossweb

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	log "github.com/sirupsen/logrus"
)

type webEngine struct {
	oss *ossClient
}

type defaultAck struct {
	Code    int32       `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Req     interface{} `json:"req"`
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
	config := cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"authorization", "content-type"},
	}
	r.Use(cors.New(config))

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
			log.Error(err)
			ctx.JSON(http.StatusUnauthorized, defaultAck{
				Code:    int32(ERROR_UNAUTHORIZED),
				Message: err.Error(),
				Req:     req,
			})
			return
		}

		if err := ctx.ShouldBindJSON(&req); err != nil {
			log.Error(err)
			ctx.JSON(http.StatusBadRequest, defaultAck{
				Code:    int32(ERROR_BADARGS),
				Message: "invalid json: " + err.Error(),
				Req:     req,
			})
			return
		}

		list, err := web.oss.List(auth.Username, req.Path)
		if err != nil {
			log.Error(err)
			ctx.JSON(http.StatusBadRequest, defaultAck{
				Code:    3,
				Message: err.Error(),
				Req:     req,
			})
			return
		}
		ctx.JSON(http.StatusOK, defaultAck{
			Code:    int32(ERROR_SUCCESS),
			Message: "success",
			Data:    list,
			Req:     req,
		})

	})

	r.POST("/api/geturl", func(ctx *gin.Context) {
		type listReq struct {
			Path string `json:"path"`
		}
		var req listReq

		auth, err := web.tokenAuth(ctx)
		if err != nil {
			log.Error(err)
			ctx.JSON(http.StatusUnauthorized, defaultAck{
				Code:    int32(ERROR_UNAUTHORIZED),
				Message: err.Error(),
				Req:     req,
			})
			return
		}

		if err := ctx.ShouldBindJSON(&req); err != nil {
			log.Error(err)
			ctx.JSON(http.StatusBadRequest, defaultAck{
				Code:    int32(ERROR_BADARGS),
				Message: "invalid json: " + err.Error(),
				Req:     req,
			})
			return
		}
		url, err := web.oss.GetSignUrl(auth.Username, req.Path)
		if err != nil {
			log.Error(err)
			ctx.JSON(http.StatusBadRequest, defaultAck{
				Code:    5,
				Message: "oss error: " + err.Error(),
				Req:     req,
			})
			return
		}
		ctx.JSON(http.StatusOK, defaultAck{
			Code:    0,
			Message: "success",
			Data:    url,
			Req:     req,
		})
	})

	r.POST("/api/get", func(ctx *gin.Context) {
		type listReq struct {
			Path string `json:"path"`
		}
		var req listReq

		auth, err := web.tokenAuth(ctx)
		if err != nil {
			log.Error(err)
			ctx.JSON(http.StatusUnauthorized, defaultAck{
				Code:    int32(ERROR_UNAUTHORIZED),
				Message: err.Error(),
				Req:     req,
			})
			return
		}

		if err := ctx.ShouldBindJSON(&req); err != nil {
			log.Error(err)
			ctx.JSON(http.StatusBadRequest, defaultAck{
				Code:    int32(ERROR_BADARGS),
				Message: "invalid json: " + err.Error(),
				Req:     req,
			})
			return
		}

		header, data, err := web.oss.Get(auth.Username, req.Path)
		if err != nil {
			log.Error(err)
			ctx.JSON(http.StatusBadRequest, defaultAck{
				Code:    3,
				Message: err.Error(),
				Req:     req,
			})
			return
		}

		// 两种下载方案
		if true {
			contentLength, err := strconv.ParseInt(header.Get("ContentLength"), 10, 64)
			if err != nil {
				panic(err)
			}
			contentType := header.Get("Content-Type")
			extraHeaders := map[string]string{
				"Content-Disposition": `attachment;"`,
			}
			ctx.DataFromReader(http.StatusOK, contentLength, contentType, data, extraHeaders)
		} else {
			var byteData []byte
			byteData, err = io.ReadAll(data)
			if err != nil {
				log.Error(err)
				ctx.JSON(http.StatusBadRequest, defaultAck{
					Code:    3,
					Message: err.Error(),
					Req:     req,
				})
				return
			}
			ctx.Data(200, "application/octet-stream", byteData)
		}

	})

	r.POST("/api/put", func(ctx *gin.Context) {
		auth, err := web.tokenAuth(ctx)
		if err != nil {
			log.Error(err)
			ctx.JSON(http.StatusUnauthorized, defaultAck{
				Code:    int32(ERROR_UNAUTHORIZED),
				Message: err.Error(),
			})
			return
		}

		file, err := ctx.FormFile("file")
		if err != nil {
			log.Error(err)
			ctx.JSON(http.StatusBadRequest, defaultAck{
				Code:    int32(ERROR_BADARGS),
				Message: fmt.Sprintf("get data error %v", err),
			})
			return
		}
		if file.Filename == "" {
			log.Error(err)
			ctx.JSON(http.StatusBadRequest, defaultAck{
				Code:    int32(ERROR_BADARGS),
				Message: err.Error(),
			})
			return
		}
		paths := strings.Split(filepath.ToSlash(file.Filename), "/")
		if paths[len(paths)-1] == "" {
			log.Error(err)
			ctx.JSON(http.StatusBadRequest, defaultAck{
				Code:    int32(ERROR_BADARGS),
				Message: err.Error(),
			})
			return
		}
		src, err := file.Open()
		if err != nil {
			log.Error(err)
			ctx.JSON(http.StatusBadRequest, defaultAck{
				Code:    2,
				Message: err.Error(),
			})
			return
		}
		defer src.Close()
		err = web.oss.Put(auth.Username, filepath.ToSlash(file.Filename), src)
		if err != nil {
			log.Error(err)
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
		return errors.New("not found api port info in config")
	}
	r.Run(fmt.Sprintf("0.0.0.0:%v", apiPort)) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
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
