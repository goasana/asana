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
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/elazarl/go-bindata-assetfs"
	"github.com/goasana/framework/testdata"
)

var header = `{{define "header"}}
<h1>Hello, asana!</h1>
{{end}}`

var index = `<!DOCTYPE html>
<html>
  <head>
    <title>asana welcome template</title>
  </head>
  <body>
{{template "block"}}
{{template "header"}}
{{template "blocks/block.tpl"}}
  </body>
</html>
`

var block = `{{define "block"}}
<h1>Hello, blocks!</h1>
{{end}}`

func TestTemplate(t *testing.T) {
	dir := "_asanaTmp"
	files := []string{
		"header.tpl",
		"index.tpl",
		"blocks/block.tpl",
	}
	if err := os.MkdirAll(dir, 0777); err != nil {
		t.Fatal(err)
	}
	for k, name := range files {
		_ = os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0777)
		if f, err := os.Create(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		} else {
			if k == 0 {
				_, _ = f.WriteString(header)
			} else if k == 1 {
				_, _ = f.WriteString(index)
			} else if k == 2 {
				_, _ = f.WriteString(block)
			}

			_ = f.Close()
		}
	}
	if err := AddViewPath(dir); err != nil {
		t.Fatal(err)
	}
	asanaTemplates := asanaViewPathTemplates[dir]
	if len(asanaTemplates) != 3 {
		t.Fatalf("should be 3 but got %v", len(asanaTemplates))
	}
	if err := asanaTemplates["index.tpl"].ExecuteTemplate(os.Stdout, "index.tpl", nil); err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		_ = os.RemoveAll(filepath.Join(dir, name))
	}
	_ = os.RemoveAll(dir)
}

var menu = `<div class="menu">
<ul>
<li>menu1</li>
<li>menu2</li>
<li>menu3</li>
</ul>
</div>
`
var user = `<!DOCTYPE html>
<html>
  <head>
    <title>asana welcome template</title>
  </head>
  <body>
{{template "../public/menu.tpl"}}
  </body>
</html>
`

func TestRelativeTemplate(t *testing.T) {
	dir := "_asanaTmp"

	//Just add dir to known viewPaths
	if err := AddViewPath(dir); err != nil {
		t.Fatal(err)
	}

	files := []string{
		"easyui/public/menu.tpl",
		"easyui/rbac/user.tpl",
	}
	if err := os.MkdirAll(dir, 0777); err != nil {
		t.Fatal(err)
	}
	for k, name := range files {
		_ = os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0777)
		if f, err := os.Create(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		} else {
			if k == 0 {
				_, _ = f.WriteString(menu)
			} else if k == 1 {
				_, _ = f.WriteString(user)
			}
			f.Close()
		}
	}
	if err := BuildTemplate(dir, files[1]); err != nil {
		t.Fatal(err)
	}
	asanaTemplates := asanaViewPathTemplates[dir]
	if err := asanaTemplates["easyui/rbac/user.tpl"].ExecuteTemplate(os.Stdout, "easyui/rbac/user.tpl", nil); err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		_ = os.RemoveAll(filepath.Join(dir, name))
	}
	_ = os.RemoveAll(dir)
}

var add = `{{ template "layout_blog.tpl" . }}
{{ define "css" }}
        <link rel="stylesheet" href="/static/css/current.css">
{{ end}}


{{ define "content" }}
        <h2>{{ .Title }}</h2>
        <p> This is SomeVar: {{ .SomeVar }}</p>
{{ end }}

{{ define "js" }}
    <script src="/static/js/current.js"></script>
{{ end}}`

var layoutBlog = `<!DOCTYPE html>
<html>
<head>
    <title>Lin Li</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8">
    <link rel="stylesheet" href="http://netdna.bootstrapcdn.com/bootstrap/3.0.3/css/bootstrap.min.css">
    <link rel="stylesheet" href="http://netdna.bootstrapcdn.com/bootstrap/3.0.3/css/bootstrap-theme.min.css">
     {{ block "css" . }}{{ end }}
</head>
<body>

    <div class="container">
        {{ block "content" . }}{{ end }}
    </div>
    <script type="text/javascript" src="http://code.jquery.com/jquery-2.0.3.min.js"></script>
    <script src="http://netdna.bootstrapcdn.com/bootstrap/3.0.3/js/bootstrap.min.js"></script>
     {{ block "js" . }}{{ end }}
</body>
</html>`

var output = `<!DOCTYPE html>
<html>
<head>
    <title>Lin Li</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8">
    <link rel="stylesheet" href="http://netdna.bootstrapcdn.com/bootstrap/3.0.3/css/bootstrap.min.css">
    <link rel="stylesheet" href="http://netdna.bootstrapcdn.com/bootstrap/3.0.3/css/bootstrap-theme.min.css">
     
        <link rel="stylesheet" href="/static/css/current.css">

</head>
<body>

    <div class="container">
        
        <h2>Hello</h2>
        <p> This is SomeVar: val</p>

    </div>
    <script type="text/javascript" src="http://code.jquery.com/jquery-2.0.3.min.js"></script>
    <script src="http://netdna.bootstrapcdn.com/bootstrap/3.0.3/js/bootstrap.min.js"></script>
     
    <script src="/static/js/current.js"></script>

</body>
</html>





`

func TestTemplateLayout(t *testing.T) {
	dir := "_asanaTmp"
	files := []string{
		"add.tpl",
		"layout_blog.tpl",
	}
	if err := os.MkdirAll(dir, 0777); err != nil {
		t.Fatal(err)
	}
	for k, name := range files {
		_ = os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0777)
		if f, err := os.Create(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		} else {
			if k == 0 {
				_, _ = f.WriteString(add)
			} else if k == 1 {
				_, _ = f.WriteString(layoutBlog)
			}
			_ = f.Close()
		}
	}
	if err := AddViewPath(dir); err != nil {
		t.Fatal(err)
	}
	asanaTemplates := asanaViewPathTemplates[dir]
	if len(asanaTemplates) != 2 {
		t.Fatalf("should be 2 but got %v", len(asanaTemplates))
	}
	out := bytes.NewBufferString("")
	if err := asanaTemplates["add.tpl"].ExecuteTemplate(out, "add.tpl", map[string]string{"Title": "Hello", "SomeVar": "val"}); err != nil {
		t.Fatal(err)
	}
	if out.String() != output {
		t.Log(out.String())
		t.Fatal("Compare failed")
	}
	for _, name := range files {
		_ = os.RemoveAll(filepath.Join(dir, name))
	}
	_ = os.RemoveAll(dir)
}

type TestingFileSystem struct {
	assetfs *assetfs.AssetFS
}

func (d TestingFileSystem) Open(name string) (http.File, error) {
	return d.assetfs.Open(name)
}

var outputBinData = `<!DOCTYPE html>
<html>
  <head>
    <title>asana welcome template</title>
  </head>
  <body>

	
<h1>Hello, blocks!</h1>

	
<h1>Hello, asana!</h1>

	

	<h2>Hello</h2>
	<p> This is SomeVar: val</p>
  </body>
</html>
`

func TestFsBinData(t *testing.T) {
	SetTemplateFSFunc(func() http.FileSystem {
		return TestingFileSystem{&assetfs.AssetFS{Asset: testdata.Asset, AssetDir: testdata.AssetDir, AssetInfo: testdata.AssetInfo}}
	})
	dir := "views"
	if err := AddViewPath("views"); err != nil {
		t.Fatal(err)
	}
	asanaTemplates := asanaViewPathTemplates[dir]
	if len(asanaTemplates) != 3 {
		t.Fatalf("should be 3 but got %v", len(asanaTemplates))
	}
	if err := asanaTemplates["index.tpl"].ExecuteTemplate(os.Stdout, "index.tpl", map[string]string{"Title": "Hello", "SomeVar": "val"}); err != nil {
		t.Fatal(err)
	}
	out := bytes.NewBufferString("")
	if err := asanaTemplates["index.tpl"].ExecuteTemplate(out, "index.tpl", map[string]string{"Title": "Hello", "SomeVar": "val"}); err != nil {
		t.Fatal(err)
	}

	if out.String() != outputBinData {
		t.Log(out.String())
		t.Fatal("Compare failed")
	}
}
