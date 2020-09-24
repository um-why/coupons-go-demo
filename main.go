package main

import (
	"coupons-go-demo/manage"
	"github.com/gin-gonic/gin"
	"github.com/hprose/hprose-golang/rpc"
	"strconv"
)

type casInfo struct {
	Errcode  int8
	Errmsg   string
	Platform string
	User     CasUser
}

type CasUser struct {
	Id       int32
	Store_id int32
}

type checkTicket struct {
	CheckTicket func(string) (casInfo, error)
}

func ScenterRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		ticket := c.GetHeader("ticket")
		if ticket == "" {
			c.AbortWithStatusJSON(433, gin.H{
				"errcode": 433,
				"errmsg":  "登录状态错误",
			})
			return
		}
		var checkTicket *checkTicket
		client := rpc.NewClient("tcp://192.168.56.3:10000")
		client.UseService(&checkTicket)
		checkRs, err := checkTicket.CheckTicket(ticket)
		if err != nil {
			c.AbortWithStatusJSON(433, gin.H{
				"errcode": 433,
				"errmsg":  "登录状态错误.",
			})
			return
		}
		if checkRs.Errcode != 0 {
			c.AbortWithStatusJSON(433, gin.H{
				"errcode": 433,
				"errmsg":  "登录状态错误:" + checkRs.Errmsg,
			})
			return
		}
		if checkRs.Platform != "store" {
			c.AbortWithStatusJSON(433, gin.H{
				"errcode": 433,
				"errmsg":  "不是商家用户角色",
			})
			return
		}
		c.Params = append(c.Params, gin.Param{
			Key:   "store_id",
			Value: strconv.Itoa(int(checkRs.User.Store_id)),
		})
		c.Next()
	}
}

func main() {
	router := gin.New()

	scenter := router.Group("/")
	scenter.Use(ScenterRole())
	{
		scenter.POST("/manage/coupon/add", manage.Add)
		scenter.GET("/manage/coupon/list", manage.GetLists)
		scenter.GET("/manage/coupon/id/:id", manage.GetDetail)
		scenter.POST("/manage/coupon/id/:id", manage.Edit)
		scenter.PUT("/manage/coupon/:id/operation", manage.Operation)
		scenter.DELETE("/manage/coupon/id/:id", manage.Delete)
		scenter.GET("/manage/coupon/check/:code", manage.GetByCheckCode)
		scenter.POST("/manage/coupon/check/:code", manage.CheckSale)
		scenter.GET("/manage/coupon/id/:id/statistics", manage.Statistics)
		scenter.GET("/manage/coupon/id/:id/receive", manage.Receive)
	}

	router.Run(":8080")
}
