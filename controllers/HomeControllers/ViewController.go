package HomeControllers

import (
	"fmt"

	"strings"

	"time"

	"github.com/TruthHun/DocHub/helper"
	"github.com/TruthHun/DocHub/models"
	"github.com/astaxie/beego/orm"
)

type ViewController struct {
	BaseController
}

func (this *ViewController) Get() {
	fmt.Print("开始执行浏览程序\n")
	id, _ := this.GetInt(":id")
	fmt.Print("输出id为:\n")
	fmt.Print(id)
	fmt.Print("判断id是否小于1开始\n")
	if id < 1 {
		this.Redirect("/", 302)
		return
		fmt.Print("判断id是否小于1中。。。。\n")
	}
	fmt.Print("判断id是否小于1结束\n")

	doc, rows, err := models.NewDocument().GetById(id)
	fmt.Print("输出doc,rows，err信息\n")
	fmt.Print(doc)
	fmt.Print("\n")
	fmt.Print(rows)
	fmt.Print("\n")
	fmt.Print(err)
	fmt.Print("判断err是否为不等于空\n")
	if err != nil || rows != 1 {
		this.Abort("404")
	}
	//文档已被删除
	fmt.Print("判断文档是否被删除\n")
	if fmt.Sprintf("%v", doc["Status"]) == "-1" {
		this.Abort("404")
	}

	var chanelTitle, parentTitle, childrenTitle interface{}

	breadcrumb, _, _ := models.GetList(models.GetTableCategory(), 1, 3, orm.NewCondition().And("Id__in", doc["Cid"], doc["ChanelId"], doc["Pid"]))
	fmt.Print("输出breadcrumb：\n")
	fmt.Print(breadcrumb)
	fmt.Print("输出breadcrumb结束\n")
	for _, v := range breadcrumb {
		switch fmt.Sprintf("%v", v["Id"]) {
		case fmt.Sprintf("%v", doc["ChanelId"]):
			this.Data["CrumbChanel"] = v
			chanelTitle = v["Title"]
			this.Data["Chanel"] = v["Alias"]
		case fmt.Sprintf("%v", doc["Pid"]):
			this.Data["CrumbParent"] = v
			parentTitle = v["Title"]
		case fmt.Sprintf("%v", doc["Cid"]):
			childrenTitle = v["Title"]
			//热门文档，根据当前所属分类去获取
			TimeStart := int(time.Now().Unix()) - this.Sys.TimeExpireHotspot
			//热门文档
			this.Data["Hots"], _, _ = models.NewDocument().SimpleList(fmt.Sprintf("di.Cid=%v and di.TimeCreate>%v", doc["Cid"], TimeStart), 10, "Dcnt")
			//最新文档
			this.Data["News"], _, _ = models.NewDocument().SimpleList(fmt.Sprintf("di.Cid=%v", doc["Cid"]), 10, "Id")
			this.Data["CrumbChildren"] = v
		}
	}
	fmt.Print("打印表中文档信息开始\n")
	models.Regulate(models.GetTableDocumentInfo(), "Vcnt", 1, "`Id`=?", id)
	fmt.Print("打印表中文档信息结束\n")
	this.Data["PageId"] = "wenku-content"
	this.Data["Doc"] = doc
	pages := helper.Interface2Int(doc["Page"])
	PageShow := 5
	if pages > PageShow {
		this.Data["PreviewPages"] = make([]string, PageShow)
	} else {
		this.Data["PreviewPages"] = make([]string, pages)
	}
	this.Data["TotalPages"] = pages
	this.Data["PageShow"] = PageShow
	fmt.Print(this.Data)
	if this.Data["Comments"], _, err = models.NewDocumentComment().GetCommentList(id, 1, 10); err != nil {
		helper.Logger.Error(err.Error())
	}
	seoTitle := fmt.Sprintf("[%v·%v·%v] ", chanelTitle, parentTitle, childrenTitle) + doc["Title"].(string)
	seoKeywords := fmt.Sprintf("%v,%v,%v,", chanelTitle, parentTitle, childrenTitle) + doc["Keywords"].(string)
	seoDesc := doc["Description"].(string)
	this.Data["Seo"] = models.NewSeo().GetByPage("PC-View", seoTitle, seoKeywords, seoDesc, this.Sys.Site)
	this.Xsrf()
	ext := fmt.Sprintf("%v", doc["Ext"])
	this.Data["Reasons"] = models.NewSys().GetReportReasons()
	if pages == 0 && (ext == "txt" || ext == "chm" || ext == "umd" || ext == "epub" || ext == "mobi") {
		this.Data["OnlyCover"] = true
		//不能预览的文档
		this.TplName = "disabled.html"
	} else {
		this.TplName = "svg.html"
	}

}

//文档下载
func (this *ViewController) Download() {
	id, _ := this.GetInt(":id")
	if id > 0 {
		if this.IsLogin > 0 {
			info, rows, err := models.NewDocument().GetById(id)
			if err != nil {
				helper.Logger.Error(err.Error())
			}
			fmt.Print("开始下载文档......\n")
			if rows > 0 {
				if helper.Interface2Int(info["Status"]) != -1 { //文档未被删除
					//下载需要的金币[注意：price的值是负值，表示扣除金币]
					price := -helper.Interface2Int(info["Price"])
					free := models.NewFreeDown().IsFreeDown(this.IsLogin, id)
					if free.Id > 0 {
						if free.TimeCreate > int(time.Now().Unix())-this.Sys.FreeDay*24*3600 { //免费下载期限内
							price = 0
						}
					}
					if userinfo := models.NewUser().UserInfo(this.IsLogin); userinfo.Coin >= price {
						//扣除金币
						models.Regulate(models.GetTableUserInfo(), "Coin", price, fmt.Sprintf("Id=%v", info["Uid"]))
						logs := models.CoinLog{
							Uid:  this.IsLogin,
							Coin: price,
							Log:  fmt.Sprintf("下载文档(%v)，消耗 %v 个金币", info["Title"], price),
						}
						models.NewCoinLog().LogRecord(logs)
						if price < 0 { //分享文档的用户金币增加
							models.Regulate(models.GetTableUserInfo(), "Coin", -price, fmt.Sprintf("Id=%v", info["Uid"]))
							logs = models.CoinLog{
								Uid:  helper.Interface2Int(info["Uid"]),
								Coin: -price,
								Log:  fmt.Sprintf("文档(%v)被下载，获得 %v 个金币", info["Title"], -price),
							}
							models.NewCoinLog().LogRecord(logs)
						}
						file := fmt.Sprintf("%v.%v", info["Md5"], info["Ext"])
						fmt.Print(file)
						fmt.Print("aaaaaaaaaaaaaa\n")
						//设置附件名
						models.NewOss().SetObjectMeta(file, fmt.Sprintf("%v.%v", info["Title"], info["Ext"]))
						//链接签名
						url := models.NewOss().BuildSign(file)
						fmt.Print(url)

						//文档下载次数+1
						models.Regulate(models.GetTableDocumentInfo(), "Dcnt", 1, fmt.Sprintf("Id=%v", info["Id"]))
						if price < 0 { //扣除了金币，则下载可以免费下载
							if free.Id > 0 { //上次已经下载过该文档，但是过了免费期限了
								models.UpdateByIds(models.GetTableFreeDown(), "TimeCreate", time.Now().Unix(), free.Id) //更新
							} else { //插入
								var freedoc = models.FreeDown{Uid: this.IsLogin, Did: id, TimeCreate: int(time.Now().Unix())}
								orm.NewOrm().Insert(&freedoc)
							}
						}

						this.ResponseJson(true, "下载链接获取成功", map[string]interface{}{"url": url})
					} else {
						this.ResponseJson(false, "您的金币余额不足，请通过签到或者分享文档，增加您的金币财富。")
					}
				} else {
					this.ResponseJson(false, "您要下载的文档不存在")
				}
			} else {
				this.ResponseJson(false, "您要下载的文档不存在")
			}

		} else {
			this.ResponseJson(false, "请先登录")
		}
	} else {
		this.ResponseJson(false, "参数不正确")
	}
}

//是否可以免费下载
func (this *ViewController) DownFree() {
	if this.IsLogin > 0 {
		did, _ := this.GetInt("id")
		if free := models.NewFreeDown().IsFreeDown(this.IsLogin, did); free.Id > 0 && free.TimeCreate > int(time.Now().Unix())-this.Sys.FreeDay*24*3600 {
			this.ResponseJson(true, fmt.Sprintf("您上次下载过当前文档，且仍在免费下载有效期(%v天)内，本次下载免费", this.Sys.FreeDay))
		}
	}
	this.ResponseJson(false, "不能免费下载，不在免费下载期限内")
}

//文档评论
func (this *ViewController) Comment() {
	id, _ := this.GetInt(":id")
	score, _ := this.GetInt("Score")
	answer := this.GetString("Answer")
	if answer != this.Sys.Answer {
		this.ResponseJson(false, "请输入正确的答案")
	}
	if id > 0 {
		if this.IsLogin > 0 {
			if score < 1 || score > 5 {
				this.ResponseJson(false, "请给文档评分")
			} else {
				comment := models.DocumentComment{
					Uid:        this.IsLogin,
					Did:        id,
					Content:    this.GetString("Comment"),
					TimeCreate: int(time.Now().Unix()),
					Status:     true,
					Score:      score * 10000,
				}
				cnt := strings.Count(comment.Content, "") - 1
				if cnt > 255 || cnt < 8 {
					this.ResponseJson(false, "评论内容限8-255个字符")
				} else {
					_, err := orm.NewOrm().Insert(&comment)
					if err != nil {
						this.ResponseJson(false, "发表评论失败：每人仅限给每个文档点评一次")
					} else {
						//文档评论人数增加
						sql := fmt.Sprintf("UPDATE `%v` SET `Score`=(`Score`*`ScorePeople`+%v)/(`ScorePeople`+1),`ScorePeople`=`ScorePeople`+1 WHERE Id=%v", models.GetTableDocumentInfo(), comment.Score, comment.Did)
						_, err := orm.NewOrm().Raw(sql).Exec()
						if err != nil {
							helper.Logger.Error(err.Error())
						}
						this.ResponseJson(true, "恭喜您，评论发表成功")
					}
				}
			}
		} else {
			this.ResponseJson(false, "评论失败，您当前处于未登录状态，请先登录")
		}
	} else {
		this.ResponseJson(false, "评论失败，参数不正确")
	}
}

//获取评论列表
func (this *ViewController) GetComment() {
	p, _ := this.GetInt("p", 1)
	did, _ := this.GetInt("did")
	if p > 0 && did > 0 {
		if rows, _, err := models.NewDocumentComment().GetCommentList(did, p, 10); err != nil {
			helper.Logger.Error(err.Error())
			this.ResponseJson(false, "评论列表获取失败")
		} else {
			this.ResponseJson(true, "评论列表获取成功", rows)
		}
	}
}
