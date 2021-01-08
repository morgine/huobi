package routes

import (
	"github.com/gin-gonic/gin"
	config2 "github.com/morgine/pkg/config"
	"github.com/shopspring/decimal"
	"huobi/config"
	"huobi/flow"
	"huobi/model"
)

func RegisterRoutes(engine *gin.Engine, configs config2.Configs) []*flow.Client {

	cfg := &config.Server{}
	err := configs.UnmarshalSub("server", cfg)
	if err != nil {
		panic(err)
	}

	subscribes := cfg.InitSubscribes()
	engine.GET("/subscribes", func(ctx *gin.Context) {
		ctx.JSON(200, subscribes)
	})

	var clients []*flow.Client
	for _, subscribe := range subscribes {
		gorm, err := config.NewPostgresORM("postgres", "gorm", subscribe.Symbol, configs)
		if err != nil {
			panic(err)
		}
		db := model.NewDB(gorm)

		client := flow.NewClient(subscribe.ClientId, subscribe.Symbol, 10)

		client.Listen([]int64{10, 30, 60, 300, 900, 3600, 14400}, func(price decimal.Decimal, sectionGetter flow.SectionGetter) {
			var section = &model.Section{
				ID:          0,
				Buy10:       0,
				Sell10:      0,
				Inflow10:    0,
				Buy30:       0,
				Sell30:      0,
				Inflow30:    0,
				Buy60:       0,
				Sell60:      0,
				Inflow60:    0,
				Buy300:      0,
				Sell300:     0,
				Inflow300:   0,
				Buy900:      0,
				Sell900:     0,
				Inflow900:   0,
				Buy3600:     0,
				Sell3600:    0,
				Inflow3600:  0,
				Buy14400:    0,
				Sell14400:   0,
				Inflow14400: 0,
				EndTime:     0,
			}
			s := sectionGetter.GetSection(10)
			section.Buy10 = s.Buy
			section.Sell10 = s.Sell
			section.Inflow10 = s.Inflow
			section.EndTime = s.EndTime
			s = sectionGetter.GetSection(30)
			section.Buy30 = s.Buy
			section.Sell30 = s.Sell
			section.Inflow30 = s.Inflow
			s = sectionGetter.GetSection(60)
			section.Buy60 = s.Buy
			section.Sell60 = s.Sell
			section.Inflow60 = s.Inflow
			s = sectionGetter.GetSection(300)
			section.Buy300 = s.Buy
			section.Sell300 = s.Sell
			section.Inflow300 = s.Inflow
			s = sectionGetter.GetSection(900)
			section.Buy900 = s.Buy
			section.Sell900 = s.Sell
			section.Inflow900 = s.Inflow
			s = sectionGetter.GetSection(3600)
			section.Buy3600 = s.Buy
			section.Sell3600 = s.Sell
			section.Inflow3600 = s.Inflow
			s = sectionGetter.GetSection(14400)
			section.Buy14400 = s.Buy
			section.Sell14400 = s.Sell
			section.Inflow14400 = s.Inflow
			db.CreateSection(section)
		})

		clients = append(clients, client)

		engine.GET("/count-sections-"+subscribe.Symbol, func(ctx *gin.Context) {
			ctx.JSON(200, db.CountSection())
		})

		{
			type params struct {
				Limit, Offset int
			}
			engine.GET("/sections-"+subscribe.Symbol, func(ctx *gin.Context) {
				ps := &params{}
				err := ctx.Bind(&ps)
				if err != nil {
					ctx.Error(err)
				} else {
					ctx.JSON(200, db.FindSections(ps.Limit, ps.Offset))
				}
			})
		}
	}
	return clients
}
