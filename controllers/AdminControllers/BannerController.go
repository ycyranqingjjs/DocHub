package AdminControllers

import (
	"fmt"

	"time"

	"os"

	"strings"

	"github.com/TruthHun/DocHub/helper"
	"github.com/TruthHun/DocHub/models"
	"github.com/astaxie/beego/orm"
)

//IT文库注册会员管理

type BannerController struct {
	BaseController
}

//横幅列表
func (this *BannerController) Get() {
	var err error
	if this.Data["Banners"], _, err = models.NewBanner().List(1, 100); err != nil && err != orm.ErrNoRows {
		helper.Logger.Error(err.Error())
	}
	this.Data["IsBanner"] = true
	this.TplName = "index.html"
}

//新增横幅
func (this *BannerController) Add() {
	f, h, err := this.GetFile("Picture")
	if err == nil {
		defer f.Close()
		//dir := "uploads/" + time.Now().Format("2006-01-02")
		dir := "static/banner/"
		os.MkdirAll(dir, 0777)
		ext := helper.GetSuffix(h.Filename, ".")
		filepath := dir + helper.MyMD5(fmt.Sprintf("%v-%v", h.Filename, time.Now().Unix())) + "." + ext
		files := helper.MyMD5(fmt.Sprintf("%v-%v", h.Filename, time.Now().Unix())) + "." + ext //添加文件作为本地访问使用
		err = this.SaveToFile("Picture", filepath)                                             // 保存位置
		fmt.Print(err)
		if err == nil {
			//if md5str, err := helper.FileMd5(filepath); err == nil {
			if md5str, err := helper.FileMd5(filepath); err == nil {
				save := md5str + "." + ext //进行了二次加密导致和本地加密后的图片名字不一致，故这里不使用此二次加密作为本地访问名字
				err = models.NewOss().MoveToOss(filepath, save, true, true)
				if err == nil {
					var banner models.Banner
					this.ParseForm(&banner)
					banner.Picture = files
					banner.TimeCreate = int(time.Now().Unix())
					banner.Status = true
					_, err = orm.NewOrm().Insert(&banner)
				}
			}
		}
	}
	if err != nil {
		helper.Logger.Error(err.Error())
		this.ResponseJson(false, err.Error())
	}
	this.ResponseJson(true, "横幅添加成功")
}

//删除横幅
func (this *BannerController) Del() {
	var err error
	id := this.GetString("id")
	ids := strings.Split(id, ",")
	if len(ids) > 0 {
		//之所以这么做，是因为如果没有第一个参数，则参数编程了[]string，而不是[]interface{},有疑问可以自己验证试下
		if _, err = models.NewBanner().Del(ids[0], ids[1:]); err != nil {
			helper.Logger.Error(err.Error())
			this.ResponseJson(false, err.Error())
		}
	}
	this.ResponseJson(true, "删除成功")
}
