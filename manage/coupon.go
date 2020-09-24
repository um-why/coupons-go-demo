package manage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
	"github.com/hprose/hprose-golang/rpc"
	"github.com/go-redis/redis/v8"
)

type Coupons struct {
	Id            int32     `gorm:"primaryKey"`
	StoreId       int32     `form:"store_id"`
	Name          string    `form:"name"`
	Type          int8      `form:"type"`
	DateType      int8      `form:"date_type"`
	StartAt       time.Time `form:"start_at" time_format:"2006-1-2"`
	EndAt         time.Time `form:"end_at" time_format:"2006-1-2" `
	DateLimit     int16     `form:"date_limit"`
	UseRange      int8      `form:"use_range"`
	Discount      string    `form:"discount"`
	Total         int16     `form:"total"`
	Threshold     string    `form:"threshold"`
	UserType      int8      `form:"user_type"`
	ApicecLimit   int16     `form:"apicec_limit"`
	Describe      string    `form:"describe"`
	StatusReceive int8
	StatusOpen    int8
	Received      int16
	LogId         int32
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt
}

type CouponLog struct {
	Id          int32 `gorm:"primaryKey"`
	CouponsId   int32
	StoreId     int32
	Name        string
	Type        int8
	DateType    int8
	StartAt     time.Time
	EndAt       time.Time
	DateLimit   int16
	UseRange    int8
	Discount    string
	Total       int16
	Threshold   string
	UserType    int8
	ApicecLimit int16
	Describe    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

var DataSourceName string = "root:Ab_123456@tcp(192.168.56.3:3306)/yj_market?charset=utf8mb4&parseTime=True&loc=Local"

var Db *gorm.DB

func init() {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Microsecond,
			LogLevel:      logger.Info,
			//LogLevel: logger.Error,
			Colorful: false,
		},
	)
	var err error
	Db, err = gorm.Open(mysql.Open(DataSourceName), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "mr_",
			SingularTable: true,
		},
		Logger: newLogger,
	})
	if err != nil {
		panic(err)
	}
}

func Add(c *gin.Context) {
	defer errMsg(c)
	var coupon Coupons
	if c.ShouldBind(&coupon) != nil {
		fmt.Println("nil error")
	}
	_storeId, _ := strconv.Atoi(c.Param("store_id"))
	coupon.StoreId = int32(_storeId)
	checkCouponParam(coupon, false)

	openNum := getOpenTotal(coupon.StoreId)
	if openNum >= 5 {
		panic("店铺有效的卡券数量不能超过5个")
	}
	totalNum := getStoreTotalCoupon(coupon.StoreId, 0)
	if totalNum+int(coupon.Total) > 999999 {
		panic("店铺发行的卡券总数量不能超过999999张")
	}

	var couponLog CouponLog
	couponLog.StoreId = coupon.StoreId
	couponLog.Name = coupon.Name
	couponLog.Type = coupon.Type
	couponLog.DateType = coupon.DateType
	couponLog.StartAt = coupon.StartAt
	couponLog.EndAt = coupon.EndAt
	couponLog.DateLimit = coupon.DateLimit
	couponLog.UseRange = coupon.UseRange
	couponLog.Discount = coupon.Discount
	couponLog.Total = coupon.Total
	couponLog.Threshold = coupon.Threshold
	couponLog.UserType = coupon.UserType
	couponLog.ApicecLimit = coupon.ApicecLimit
	couponLog.Describe = coupon.Describe
	result := Db.Create(&couponLog)
	if result.Error != nil {
		panic("卡券记录创建错误")
	}

	coupon.StatusReceive = 1
	coupon.StatusOpen = 1
	coupon.Received = 0
	coupon.LogId = couponLog.Id
	result = Db.Create(&coupon)
	if result.Error != nil {
		panic("卡券信息创建错误")
	}

	result = Db.Model(&couponLog).Update("coupons_id", coupon.Id)
	if result.Error != nil {
		panic("卡券记录更新错误")
	}

	c.JSON(200, gin.H{
		"errcode": 0,
		"errmsg":  "succ",
	})
}

func checkCouponParam(param Coupons, isEdit bool) bool {
	if param.StoreId <= 0 {
		panic("参数错误1")
	}
	count := utf8.RuneCountInString(param.Name)
	if count < 2 || count > 10 {
		panic("参数错误2")
	}
	if param.Type != 1 && param.Type != 2 {
		panic("参数错误3")
	}
	if param.DateType != 1 && param.DateType != 2 {
		panic("参数错误4")
	}
	if param.DateType == 1 {
		//loc, _ := time.LoadLocation("Local")
		//startAt, _ := time.ParseInLocation("2006-1-2", param.StartAt, loc)
		//endAt, _ := time.ParseInLocation("2006-1-2", param.EndAt, loc)
		if isEdit == false && param.StartAt.Unix() < (time.Now().Unix()-86400) {
			panic("参数错误5")
		}
		if param.StartAt.Unix() > param.EndAt.Unix() || (param.EndAt.Unix()-param.StartAt.Unix()) > 365*86400 {
			panic("参数错误5")
		}
	} else {
		if param.DateLimit <= 0 || param.DateLimit > 365 {
			panic("参数错误6")
		}
	}
	if param.UseRange != 1 && param.UseRange != 2 {
		panic("参数错误7")
	}
	discount, _ := strconv.ParseFloat(param.Discount, 32)
	if param.Type == 1 {
		if discount < 0 || discount > 99999 {
			panic("参数错误8")
		}
	} else {
		if discount < 0.1 || discount > 9.9 {
			panic("参数错误8")
		}
	}
	if param.Total <= 0 || param.Total > 9999 {
		panic("参数错误9")
	}
	threshold, _ := strconv.ParseFloat(param.Threshold, 32)
	if param.Type == 1 {
		if threshold > discount {
			panic("参数错误10")
		}
	} else {
		if threshold < 0 {
			panic("参数错误10")
		}
	}
	if threshold > 999999 {
		panic("参数错误11")
	}
	if param.UserType != 1 && param.UserType != 2 && param.UserType != 3 {
		panic("参数错误12")
	}
	if param.ApicecLimit <= 0 {
		panic("参数错误13")
	}
	if param.ApicecLimit > param.Total {
		panic("参数错误14")
	}
	count = utf8.RuneCountInString(param.Describe)
	if count < 4 || count > 100 {
		panic("参数错误15")
	}
	return true
}

func getOpenTotal(storeId int32) int8 {
	sql := "select count(1) as total from mr_coupons where "
	sql += "store_id = ? and status_receive = 1"
	sql += " and deleted_at is null"
	var total int
	Db.Raw(sql, storeId).Scan(&total)
	return int8(total)
}

func getStoreTotalCoupon(storeId int32, notId int32) int {
	sql := "select sum(total) as sum from mr_coupons where "
	sql += "store_id = ?"
	if notId > 0 {
		sql += " and id <> " + strconv.Itoa(int(notId))
	}
	var sum int
	Db.Raw(sql, storeId).Scan(&sum)
	return sum
}

func errMsg(c *gin.Context) {
	if err := recover(); err != nil {
		c.JSON(200, gin.H{
			"errcode": 1,
			"errmsg":  fmt.Sprintf("%s", err),
		})
	}
}

type fmtCouponsList struct {
	Id            int32     `json:"id"`
	Name          string    `json:"name"`
	Type          int8      `json:"type"`
	TypeName      string    `json:"type_name"`
	DateType      int8      `json:"date_type"`
	StartAt       time.Time `json:"start_at"`
	EndAt         time.Time `json:"end_at"`
	DateLimit     int16     `json:"date_limit"`
	Discount      string    `json:"discount"`
	Total         int16     `json:"total"`
	Threshold     string    `json:"threshold"`
	ApicecLimit   int16     `json:"apicec_limit"`
	StatusReceive int8      `json:"status_receive"`
	StatusOpen    int8      `json:"status_open"`
	Received      int16     `json:"received"`
	StatusName    string    `json:"status_name"`
}

func GetLists(c *gin.Context) {
	storeId, _ := strconv.Atoi(c.Param("store_id"))
	if storeId <= 0 {
		storeId = 0
	}
	status, _ := strconv.Atoi(c.DefaultQuery("status", "0"))
	if status != 1 && status != 2 {
		status = 0
	}
	name := c.DefaultQuery("name", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if pageSize <= 0 {
		pageSize = 10
	}
	var coupon []Coupons
	tx := Db.Order("status_receive asc,created_at desc")
	if storeId > 0 {
		tx = tx.Where("store_id = ?", storeId)
	}
	if status > 0 {
		tx = tx.Where("status_receive = ?", status)
	}
	if name != "" {
		tx = tx.Where("name like %?%", name)
	}
	tx = tx.Limit(pageSize).Offset((page - 1) * pageSize).Find(&coupon)
	var _coupons []fmtCouponsList
	for _, item := range coupon {
		var typeName string
		if item.Type == 1 {
			typeName = "满减券"
		} else if item.Type == 2 {
			typeName = "折扣券"
		} else {
			typeName = "未知类型"
		}
		var statusName string
		statusName = "未知状态"
		if item.StatusReceive == 2 {
			statusName = "已结束"
		} else {
			if item.StatusOpen == 2 {
				statusName = "已暂停"
			} else {
				statusName = "领取中"
			}
		}

		_coupons = append(_coupons, fmtCouponsList{
			Id:            item.Id,
			Name:          item.Name,
			Type:          item.Type,
			TypeName:      typeName,
			DateType:      item.DateType,
			StartAt:       item.StartAt,
			EndAt:         item.EndAt,
			DateLimit:     item.DateLimit,
			Discount:      item.Discount,
			Total:         item.Total,
			Threshold:     item.Threshold,
			ApicecLimit:   item.ApicecLimit,
			StatusReceive: item.StatusReceive,
			StatusOpen:    item.StatusOpen,
			Received:      item.Received,
			StatusName:    statusName,
		})
	}
	c.JSON(200, gin.H{
		"errcode": 0,
		"errmsg":  "succ",
		"data":    _coupons,
	})
}

type fmtCouponsDetail struct {
	Id           int32     `json:"id"`
	Name         string    `json:"name"`
	Type         int8      `json:"type"`
	TypeName     string    `json:"type_name"`
	DateType     int8      `json:"date_type"`
	StartAt      time.Time `json:"start_at"`
	EndAt        time.Time `json:"end_at"`
	DateLimit    int16     `json:"date_limit"`
	UseRange     int8      `json:"use_range"`
	UseRangeName string    `json:"use_range_name"`
	Discount     string    `json:"discount"`
	Total        int16     `json:"total"`
	Threshold    string    `json:"threshold"`
	UserType     int8      `json:"user_type"`
	UserTypeName string    `json:"user_type_name"`
	ApicecLimit  int16     `json:"apicec_limit"`
	Describe     string    `json:"describe"`
}

func GetDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		panic("参数错误")
	}
	var coupon Coupons
	Db.First(&coupon, id)
	if coupon.Id <= 0 {
		panic("未找到")
	}
	storeId, _ := strconv.Atoi(c.Param("store_id"))
	if storeId > 0 {
		if coupon.StoreId != int32(storeId) {
			panic("未找到")
		}
	}
	var _coupons fmtCouponsDetail
	_coupons.Id = coupon.Id
	_coupons.Name = coupon.Name
	_coupons.Type = coupon.Type
	_coupons.TypeName = "未知类型"
	if coupon.Type == 1 {
		_coupons.TypeName = "满减券"
	} else if coupon.Type == 2 {
		_coupons.TypeName = "折扣券"
	}
	_coupons.DateType = coupon.DateType
	_coupons.StartAt = coupon.StartAt
	_coupons.EndAt = coupon.EndAt
	_coupons.DateLimit = coupon.DateLimit
	_coupons.UseRange = coupon.UseRange
	_coupons.UseRangeName = "未知可用"
	if coupon.UseRange == 1 {
		_coupons.UseRangeName = "全场可用"
	} else if coupon.UseRange == 2 {
		_coupons.UseRangeName = "部分可用"
	}
	_coupons.Discount = coupon.Discount
	_coupons.Total = coupon.Total
	_coupons.Threshold = coupon.Threshold
	_coupons.UserType = coupon.UserType
	_coupons.UserTypeName = "未知用户"
	if coupon.UserType == 1 {
		_coupons.UserTypeName = "全部用户"
	} else if coupon.UserType == 2 {
		_coupons.UserTypeName = "新用户"
	} else if coupon.UserType == 3 {
		_coupons.UserTypeName = "老用户"
	}
	_coupons.ApicecLimit = coupon.ApicecLimit
	_coupons.Describe = coupon.Describe
	c.JSON(200, gin.H{
		"errcode": 0,
		"errmsg":  "succ",
		"data":    _coupons,
	})
}

func Edit(c *gin.Context) {
	defer errMsg(c)
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		panic("参数错误")
	}
	storeId, _ := strconv.Atoi(c.Param("store_id"))
	if storeId <= 0 {
		panic("未找到")
	}
	var coupon Coupons
	Db.First(&coupon, id)
	if coupon.Id <= 0 {
		panic("未找到")
	}
	if storeId != int(coupon.StoreId) {
		panic("未找到")
	}
	if coupon.StatusReceive == 2 {
		panic("卡券已经发放结束，无法编辑")
	}

	var _coupon Coupons
	if c.ShouldBind(&_coupon) != nil {
		panic("参数错误")
	}
	_coupon.StoreId = int32(storeId)
	checkCouponParam(_coupon, true)
	totalNum := getStoreTotalCoupon(coupon.StoreId, coupon.Id)
	if totalNum+int(_coupon.Total) > 999999 {
		panic("店铺发行的卡券总数量不能超过999999张")
	}
	if coupon.StoreId != _coupon.StoreId {
		panic("参数错误.")
	}
	if coupon.Received >= _coupon.Total {
		panic("卡券的发行量须大于已领取的数量，请修改后重试")
	}

	var couponLog CouponLog
	couponLog.CouponsId = coupon.Id
	couponLog.StoreId = _coupon.StoreId
	couponLog.Name = _coupon.Name
	couponLog.Type = _coupon.Type
	couponLog.DateType = _coupon.DateType
	couponLog.StartAt = _coupon.StartAt
	couponLog.EndAt = _coupon.EndAt
	couponLog.DateLimit = _coupon.DateLimit
	couponLog.UseRange = _coupon.UseRange
	couponLog.Discount = _coupon.Discount
	couponLog.Total = _coupon.Total
	couponLog.Threshold = _coupon.Threshold
	couponLog.UserType = _coupon.UserType
	couponLog.ApicecLimit = _coupon.ApicecLimit
	couponLog.Describe = _coupon.Describe
	result := Db.Create(&couponLog)
	if result.Error != nil {
		panic("卡券记录修改错误")
	}

	_coupon.LogId = couponLog.Id
	Db.Model(&coupon).Updates(_coupon)
	c.JSON(200, gin.H{
		"errcode": 0,
		"errmsg":  "succ",
	})
}

func Operation(c *gin.Context) {
	defer errMsg(c)
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		panic("参数错误")
	}
	storeId, _ := strconv.Atoi(c.Param("store_id"))
	if storeId <= 0 {
		panic("未找到")
	}
	var coupon Coupons
	Db.First(&coupon, id)
	if coupon.Id <= 0 {
		panic("未找到")
	}
	if storeId != int(coupon.StoreId) {
		panic("未找到")
	}
	if coupon.StatusReceive == 2 {
		panic("卡券已经发放结束，无法编辑")
	}
	oType := c.DefaultPostForm("type", "")
	if oType != "pause" && oType != "continue" && oType != "end" {
		panic("参数错误.")
	}
	if coupon.StatusOpen == 2 && oType == "pause" {
		c.JSON(200, gin.H{
			"errcode": 0,
			"errmsg":  "succ",
		})
		return
	}
	if coupon.StatusOpen == 1 && oType == "continue" {
		c.JSON(200, gin.H{
			"errcode": 0,
			"errmsg":  "succ",
		})
		return
	}
	if coupon.StatusOpen != 2 && oType == "end" {
		panic("暂停中卡券才能结束")
	}

	if oType == "pause" {
		Db.Model(&coupon).Update("status_open", 2)
	} else if oType == "continue" {
		Db.Model(&coupon).Update("status_open", 1)
	} else {
		Db.Model(&coupon).Update("status_receive", 2)
	}
	c.JSON(200, gin.H{
		"errcode": 0,
		"errmsg":  "succ",
	})
	return
}

func Delete(c *gin.Context) {
	defer errMsg(c)
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		panic("参数错误")
	}
	storeId, _ := strconv.Atoi(c.Param("store_id"))
	if storeId <= 0 {
		panic("未找到")
	}
	var coupon Coupons
	Db.First(&coupon, id)
	if coupon.Id <= 0 {
		panic("未找到")
	}
	if storeId != int(coupon.StoreId) {
		panic("未找到")
	}
	if coupon.StatusReceive != 2 {
		panic("仅完结的卡券才能删除")
	}

	Db.Delete(&coupon)
	c.JSON(200, gin.H{
		"errcode": 0,
		"errmsg":  "succ",
	})
	return
}

type CouponUser struct {
	Id          int32 `gorm:"primaryKey"`
	UserId      int32
	StoreId     int32
	CouponId    int32
	CouponLogId int32
	CheckCode   string
	Status      int8
	StartAt     time.Time
	EndAt       time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type fmtGetByCheckCode struct {
	Id           int32     `json:"id"`
	Store        fmtStore  `json:"store"`
	Name         string    `json:"name"`
	Type         int8      `json:"type"`
	DateType     int8      `json:"date_type"`
	StartAt      time.Time `json:"start_at"`
	EndAt        time.Time `json:"end_at"`
	UseRange     int8      `json:"use_range"`
	UseRangeName string    `json:"use_range_name"`
	Discount     string    `json:"discount"`
	Threshold    string    `json:"threshold"`
	Describe     string    `json:"describe"`
	IsUsed       int8      `json:"is_used"`
	IsExpire     int8      `json:"is_expire"`
}

type fmtStore struct {
	Id    int32  `json:"id"`
	Title string `json:"title"`
	Logo  string `json:"logo"`
}

func GetByCheckCode(c *gin.Context) {
	defer errMsg(c)
	code := c.Param("code")
	storeId, _ := strconv.Atoi(c.Param("store_id"))
	if storeId <= 0 {
		panic("需要先登录")
	}
	var couponUser CouponUser
	Db.Where("check_code = ? and store_id = ?", code, storeId).First(&couponUser)
	if couponUser.Id <= 0 {
		panic("卡券未找到")
	}

	var couponLog CouponLog
	Db.Where("id = ? and coupons_id = ? and store_id = ?", couponUser.CouponLogId, couponUser.CouponId, couponUser.StoreId).First(&couponLog)
	if couponLog.Id <= 0 {
		panic("卡券未找到.")
	}

	store := getStoreInfo(couponUser.StoreId)

	var info fmtGetByCheckCode
	info.Id = couponUser.Id
	info.Store = store
	info.Name = couponLog.Name
	info.Type = couponLog.Type
	info.DateType = couponLog.DateType
	info.StartAt = couponLog.StartAt
	info.EndAt = couponLog.EndAt
	info.UseRange = couponLog.UseRange
	info.UseRangeName = "未知可用"
	if couponLog.UseRange == 1 {
		info.UseRangeName = "全场可用"
	} else if couponLog.UserType == 2 {
		info.UseRangeName = "部分可用"
	}
	info.Discount = couponLog.Discount
	info.Threshold = couponLog.Threshold
	info.Describe = couponLog.Describe
	if couponUser.Status == 2 {
		info.IsUsed = 1
	} else {
		info.IsUsed = 0
	}
	if couponUser.Status == 3 {
		info.IsExpire = 1
	} else {
		if couponUser.EndAt.Unix() < time.Now().Unix() {
			info.IsExpire = 1
		}
	}
	c.JSON(200, gin.H{
		"errcode": 0,
		"errmsg":  "succ",
		"data":    info,
	})
}

type StoreInfoByIds struct {
	GetStoreInfoByIds func(string) ([]fmtStore, error)
}

func (fs fmtStore) MarshalBinary() ([]byte, error) {
	return json.Marshal(fs)
}

func getStoreInfo(storeId int32) fmtStore {
	var store fmtStore
	if storeId <= 0 {
		return store
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     "192.168.56.3:6379",
		Password: "",
		DB:       0,
	})
	var ctx = context.Background()
	cacheKey := "MK:CM:GSI:" + strconv.Itoa(int(storeId))
	cacheRs, err := rdb.Get(ctx, cacheKey).Result()
	if err == nil && cacheRs != "" {
		err = json.Unmarshal([]byte(cacheRs), &store)
		if err == nil {
			return store
		}
	}

	var storeInfo *StoreInfoByIds
	client := rpc.NewClient("tcp://192.168.56.3:10001")
	client.UseService(&storeInfo)
	storeRs, err := storeInfo.GetStoreInfoByIds(strconv.Itoa(int(storeId)))
	if err != nil {
		return store
	}
	if storeRs[0].Id <= 0 && storeRs[0].Id != storeId {
		return store
	}
	store = storeRs[0]
	if store.Logo == "" {
		store.Logo = "https://ores.360vrsh.com/mob/no-store-img.jpg"
	} else if strings.Index(store.Logo, "http") != 0 {
		store.Logo = "http://test360vrsh.oss-cn-qingdao.aliyuncs.com/" + store.Logo
	}

	err = rdb.Set(ctx, cacheKey, &store, time.Second*60*5).Err()
	if err != nil {
		panic("redis存储错误")
	}
	return store
}

func CheckSale(c *gin.Context) {
	defer errMsg(c)
	code := c.Param("code")
	storeId, _ := strconv.Atoi(c.Param("store_id"))
	if storeId <= 0 {
		panic("未找到")
	}
	isEnforce := c.DefaultPostForm("is_enforce", "0")
	var couponUser CouponUser
	Db.Where("check_code = ? and store_id = ?", code, storeId).First(&couponUser)
	if couponUser.Id <= 0 {
		panic("卡券未找到")
	}
	if couponUser.Status <= 0 || couponUser.Status == 2 {
		panic("一张卡券不能再次使用")
	}

	isArrive := false
	if time.Now().Unix() < couponUser.StartAt.Unix() {
		isArrive = true
	}
	if isArrive == true && isEnforce != "1" {
		panic("该卡券未到使用日期，是否继续核销？")
	}

	isExpire := false
	if couponUser.Status == 3 {
		isExpire = true
	}
	if isExpire == false {
		if couponUser.EndAt.Unix() < time.Now().Unix() {
			isExpire = true
		}
	}
	if isExpire == true && isEnforce != "1" {
		panic("该卡券已过期，是否继续核销？")
	}

	Db.Model(&couponUser).Update("status", 2)
	c.JSON(200, gin.H{
		"errcode": 0,
		"errmsg":  "succ",
	})
}

type fmtStatistics struct {
	Id    int32 `json:"id"`
	Send  int16 `json:"send"`
	Use   int16 `json:"use"`
	UnUse int16 `json:"unuse"`
}

func Statistics(c *gin.Context) {
	defer errMsg(c)
	couponId := c.Param("id")
	storeId, _ := strconv.Atoi(c.Param("store_id"))
	var coupon Coupons
	Db.Where("id = ?", couponId).First(&coupon)
	if coupon.Id <= 0 {
		panic("卡券未找到")
	}
	if storeId > 0 {
		if coupon.StoreId != int32(storeId) {
			panic("卡券未找到")
		}
	}

	var count int64
	Db.Model(&CouponUser{}).Where("coupon_id = ? and store_id = ? and status = 2", coupon.Id, coupon.StoreId).Count(&count)
	var _statistics fmtStatistics
	_statistics.Id = coupon.Id
	_statistics.Send = coupon.Received
	_statistics.Use = int16(count)
	_statistics.UnUse = int16(int(coupon.Received) - int(count))
	c.JSON(200, gin.H{
		"errcode": 0,
		"errmsg":  "succ",
		"data":    _statistics,
	})
}

type Users struct {
	Id         int32
	UserId     int32
	Nickname   string
	HeadImgUrl string
}

type fmtReceive struct {
	Id         int32     `json:"id"`
	Nickname   string    `json:"nickname"`
	HeadImgUrl string    `json:"head_img_url"`
	StatusName string    `json:"status_name"`
	CreatedAt  time.Time `json:"created_at"`
}

func Receive(c *gin.Context) {
	defer errMsg(c)
	couponId := c.Param("id")
	storeId, _ := strconv.Atoi(c.Param("store_id"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if pageSize <= 0 {
		pageSize = 10
	}
	var coupon Coupons
	Db.Where("id = ?", couponId).First(&coupon)
	if coupon.Id <= 0 {
		panic("卡券未找到")
	}
	if storeId > 0 {
		if coupon.StoreId != int32(storeId) {
			panic("卡券未找到")
		}
	}
	var couponUser []CouponUser
	tx := Db.Where("coupon_id = ? and store_id = ?", coupon.Id, coupon.StoreId).Order("id desc")
	tx = tx.Limit(pageSize).Offset((page - 1) * pageSize).Find(&couponUser)
	var _userIds []int32
	for _, item := range couponUser {
		_userIds = append(_userIds, item.UserId)
	}
	var users []Users
	if len(_userIds) > 0 {
		Db.Find(&users, _userIds)
	}
	var _recevie []fmtReceive
	for _, item := range couponUser {
		statusName := "未知状态"
		if item.Status == 2 {
			statusName = "已使用"
		} else if item.Status == 3 {
			statusName = "已过期"
		} else {
			statusName = "未使用"
		}

		var user Users
		for _, userOne := range users {
			if item.UserId != userOne.UserId {
				continue
			}
			user = userOne
			break
		}
		if user.Nickname == "" {
			user.Nickname = "无名氏"
		}
		if user.HeadImgUrl == "" {
			user.HeadImgUrl = "//ores.360vrsh.com/mob/avatar.png"
		}

		_recevie = append(_recevie, fmtReceive{
			Id:         item.Id,
			Nickname:   user.Nickname,
			HeadImgUrl: user.HeadImgUrl,
			StatusName: statusName,
			CreatedAt:  item.CreatedAt,
		})
	}
	c.JSON(200, gin.H{
		"errcode": 0,
		"errmsg":  "succ",
		"data":    _recevie,
	})
}
