# coupons-go-demo

实际项目需求，完成的gin学习练手项目；使用到:gin、gorm、go-redis、hprose-golang

#### 项目说明

- gin的练手项目，大佬路过如有兴趣请指正；菜鸟学习后如有改善处，请帮忙修正
- 基于产品狗提出的项目需求，公司技术栈php laravel 5.5、mysql aliyun-rds 5.6、redis 、hprose-php；完成后端开发后，用这个需求来完成gin的练手；部分需求，请参加 [简要需求截图](doc/prd.md)

#### 实现功能

- gin 自定义中间件：用于验证接口请求的权限
- hprose 发起RPC请求，及自定义struct接收请求结果
- 中间件数据注入gin request请求体的params中，在后续的实现中接收该值
- gorm 和自定义struct的结合
- gorm 开启sql打印
- 定义接口响应对象或列表的struct
- 实现的功能：卡券的CURD操作、卡券的核销

##### MORE

- 开发之处，通过一个go awesome找到的ORM是xorm，开发中，第一个接口添加卡券时没有遇到问题，第二个接口获取卡券列表就走不动了，各种google百度也找不到列表的实现代码；然后换用gorm。