package flow

// 数据流容器
type container struct {
	section  *Section
	duration int64
	flows    []*Flow
}

func newContainer(duration int64) *container {
	return &container{
		section:  &Section{},
		duration: duration,
		flows:    nil,
	}
}

func (c *container) push(flow *Flow) {
	// 更新数据结束时间
	c.section.EndTime = flow.Timestamp

	// 最小时间，超过最小时间的旧数据将被删除
	minTime := flow.Timestamp - c.duration

	// 叠加新数据
	c.section.Buy += flow.Buy
	c.section.Sell += flow.Sell
	c.section.Inflow += flow.Inflow

	// 将数据加入数据流
	c.flows = append(c.flows, flow)
	idx := 0
	for _, old := range c.flows {
		// 移除旧数据
		if old.Timestamp < minTime {
			c.section.Buy -= old.Buy
			c.section.Sell -= old.Sell
			c.section.Inflow -= old.Inflow
			idx++
		} else {
			// 重置数据区间起始时间
			c.section.StartTime = old.Timestamp
			break
		}
	}
	// 删除过期数据
	if idx > 0 {
		c.flows = c.flows[idx:]
	}
}

// 数据块
type Section struct {
	Buy       int64
	Sell      int64
	Inflow    int64
	StartTime int64
	EndTime   int64
}

// 数据流
type Flow struct {
	Buy       int64
	Sell      int64
	Inflow    int64
	Timestamp int64
}
