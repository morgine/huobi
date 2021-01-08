package config

import "strings"

type Server struct {
	Subscribes string `toml:"subscribes"`
}

func (c *Server) InitSubscribes() (subs []Subscribe) {
	configs := strings.Split(strings.Replace(c.Subscribes, " ", "", -1), ",")
	for _, config := range configs {
		ss := strings.Split(config, ":")
		subs = append(subs, Subscribe{
			Symbol:   ss[0],
			ClientId: ss[1],
		})
	}
	return
}

type Subscribe struct {
	Symbol   string `toml:"symbol"`
	ClientId string `toml:"client_id"`
}
