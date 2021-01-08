module huobi

go 1.15

require (
	github.com/gin-contrib/cors v1.3.1 // indirect
	github.com/gin-gonic/gin v1.6.3
	github.com/huobirdcenter/huobi_golang v0.0.0-20201231082458-10d97afd26d8
	github.com/morgine/pkg v0.0.0-20210104083822-6aaa329258a5
	github.com/shopspring/decimal v1.2.0
	go.uber.org/zap v1.15.0
	gorm.io/gorm v1.20.7
)

replace github.com/morgine/pkg => ../morgine/pkg
