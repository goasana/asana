// Copyright 2019 asana Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package asana

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/goasana/asana/context"
	"github.com/goasana/asana/logs"
	"github.com/hashicorp/golang-lru"
)

var errNotStaticRequest = errors.New("request not a static file request")

func serverStaticRouter(ctx *context.Context) {
	if ctx.Method() != "GET" && ctx.Method() != "HEAD" {
		return
	}

	forbidden, filePath, fileInfo, err := lookupFile(ctx)
	if err == errNotStaticRequest {
		return
	}

	if forbidden {
		exception("403", ctx)
		return
	}

	if filePath == "" || fileInfo == nil {
		if BConfig.RunMode == DEV {
			logs.Warn("Can't find/open the file:", filePath, err)
		}
		http.NotFound(ctx.ResponseWriter, ctx.HTTPRequest)
		return
	}
	if fileInfo.IsDir() {
		requestURL := ctx.URL()
		if requestURL[len(requestURL)-1] != '/' {
			redirectURL := requestURL + "/"
			if ctx.HTTPRequest.URL.RawQuery != "" {
				redirectURL = redirectURL + "?" + ctx.HTTPRequest.URL.RawQuery
			}
			_ = ctx.SetStatus(302).Redirect(redirectURL)
		} else {
			//serveFile will list dir
			http.ServeFile(ctx.ResponseWriter, ctx.HTTPRequest, filePath)
		}
		return
	} else if fileInfo.Size() > int64(BConfig.WebConfig.StaticCacheFileSize) {
		//over size file serve with http module
		http.ServeFile(ctx.ResponseWriter, ctx.HTTPRequest, filePath)
		return
	}

	var enableCompress = BConfig.EnableGzip && isStaticCompress(filePath)
	var acceptEncoding string
	if enableCompress {
		acceptEncoding = context.ParseEncoding(ctx.HTTPRequest)
	}
	b, n, sch, reader, err := openFile(filePath, fileInfo, acceptEncoding)
	if err != nil {
		if BConfig.RunMode == DEV {
			logs.Warn("Can't compress the file:", filePath, err)
		}
		http.NotFound(ctx.ResponseWriter, ctx.HTTPRequest)
		return
	}

	if b {
		ctx.SetHeader(context.HeaderContentEncoding, n)
	} else {
		ctx.SetHeader(context.HeaderContentLength, strconv.FormatInt(sch.size, 10))
	}

	http.ServeContent(ctx.ResponseWriter, ctx.HTTPRequest, filePath, sch.modTime, reader)
}

type serveContentHolder struct {
	data       []byte
	modTime    time.Time
	size       int64
	originSize int64 //original file size:to judge file changed
	encoding   string
}

type serveContentReader struct {
	*bytes.Reader
}

var (
	staticFileLruCache *lru.Cache
	lruLock            sync.RWMutex
)

func openFile(filePath string, fi os.FileInfo, acceptEncoding string) (bool, string, *serveContentHolder, *serveContentReader, error) {
	if staticFileLruCache == nil {
		//avoid lru cache error
		if BConfig.WebConfig.StaticCacheFileNum >= 1 {
			staticFileLruCache, _ = lru.New(BConfig.WebConfig.StaticCacheFileNum)
		} else {
			staticFileLruCache, _ = lru.New(1)
		}
	}
	mapKey := acceptEncoding + ":" + filePath
	lruLock.RLock()
	var mapFile *serveContentHolder
	if cacheItem, ok := staticFileLruCache.Get(mapKey); ok {
		mapFile = cacheItem.(*serveContentHolder)
	}
	lruLock.RUnlock()
	if isOk(mapFile, fi) {
		reader := &serveContentReader{Reader: bytes.NewReader(mapFile.data)}
		return mapFile.encoding != "", mapFile.encoding, mapFile, reader, nil
	}
	lruLock.Lock()
	defer lruLock.Unlock()
	if cacheItem, ok := staticFileLruCache.Get(mapKey); ok {
		mapFile = cacheItem.(*serveContentHolder)
	}
	if !isOk(mapFile, fi) {
		file, err := os.Open(filePath)
		if err != nil {
			return false, "", nil, nil, err
		}
		defer file.Close()
		var bufferWriter bytes.Buffer
		_, n, err := context.WriteFile(acceptEncoding, &bufferWriter, file)
		if err != nil {
			return false, "", nil, nil, err
		}
		mapFile = &serveContentHolder{data: bufferWriter.Bytes(), modTime: fi.ModTime(), size: int64(bufferWriter.Len()), originSize: fi.Size(), encoding: n}
		if isOk(mapFile, fi) {
			staticFileLruCache.Add(mapKey, mapFile)
		}
	}

	reader := &serveContentReader{Reader: bytes.NewReader(mapFile.data)}
	return mapFile.encoding != "", mapFile.encoding, mapFile, reader, nil
}

func isOk(s *serveContentHolder, fi os.FileInfo) bool {
	if s == nil {
		return false
	} else if s.size > int64(BConfig.WebConfig.StaticCacheFileSize) {
		return false
	}
	return s.modTime == fi.ModTime() && s.originSize == fi.Size()
}

// isStaticCompress detect static files
func isStaticCompress(filePath string) bool {
	for _, statExtension := range BConfig.WebConfig.StaticExtensionsToGzip {
		if strings.HasSuffix(strings.ToLower(filePath), strings.ToLower(statExtension)) {
			return true
		}
	}
	return false
}

// searchFile search the file by url path
// if none the static file prefix matches ,return notStaticRequestErr
func searchFile(ctx *context.Context) (string, os.FileInfo, error) {
	requestPath := filepath.ToSlash(filepath.Clean(ctx.HTTPRequest.URL.Path))
	// special processing : favicon.ico/robots.txt  can be in any static dir
	if requestPath == "/favicon.ico" || requestPath == "/robots.txt" {
		file := path.Join(".", requestPath)
		if fi, _ := os.Stat(file); fi != nil {
			return file, fi, nil
		}
		for _, staticDir := range BConfig.WebConfig.StaticDir {
			filePath := path.Join(staticDir, requestPath)
			if fi, _ := os.Stat(filePath); fi != nil {
				return filePath, fi, nil
			}
		}
		return "", nil, errNotStaticRequest
	}

	for prefix, staticDir := range BConfig.WebConfig.StaticDir {
		if !strings.Contains(requestPath, prefix) {
			continue
		}
		if prefix != "/" && len(requestPath) > len(prefix) && requestPath[len(prefix)] != '/' {
			continue
		}
		filePath := path.Join(staticDir, requestPath[len(prefix):])
		if fi, err := os.Stat(filePath); fi != nil {
			return filePath, fi, err
		}
	}
	return "", nil, errNotStaticRequest
}

// lookupFile find the file to serve
// if the file is dir ,search the index.html as default file( MUST NOT A DIR also)
// if the index.html not exist or is a dir, give a forbidden response depending on  DirectoryIndex
func lookupFile(ctx *context.Context) (bool, string, os.FileInfo, error) {
	fp, fi, err := searchFile(ctx)
	if fp == "" || fi == nil {
		return false, "", nil, err
	}
	if !fi.IsDir() {
		return false, fp, fi, err
	}
	if requestURL := ctx.URL(); requestURL[len(requestURL)-1] == '/' {
		ifp := filepath.Join(fp, "index.html")
		if ifi, _ := os.Stat(ifp); ifi != nil && ifi.Mode().IsRegular() {
			return false, ifp, ifi, err
		}
	}
	return !BConfig.WebConfig.DirectoryIndex, fp, fi, err
}
