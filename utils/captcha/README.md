# Captcha

an example for use captcha

```
package controllers

import (
	"github.com/goasana/framework"
	"github.com/goasana/framework/cache"
	"github.com/goasana/framework/utils/captcha"
)

var cpt *captcha.Captcha

func init() {
	// use asana cache system store the captcha data
	store := cache.NewMemoryCache()
	cpt = captcha.NewWithFilter("/captcha/", store)
}

type MainController struct {
	asana.Controller
}

func (this *MainController) Get() {
	this.TplName = "index.tpl"
}

func (this *MainController) Post() {
	this.TplName = "index.tpl"

	this.Data["Success"] = cpt.VerifyReq(this.Ctx.Request)
}
```

template usage

```
{{.Success}}
<form action="/" method="post">
	{{create_captcha}}
	<input name="captcha" type="text">
</form>
```
