package processor

import (
	"bytes"
	"errors"
	"github.com/stretchr/testify/suite"
	"strings"
	"testing"
	"time"
	"yadro-intern/cmd/config"
	"yadro-intern/internal/apierror"
	"yadro-intern/internal/model"
	"yadro-intern/internal/storage"
)

type processorTestSuite struct {
	cfg *config.Processor

	suite.Suite
}

func newProcessorTestSuite() *processorTestSuite {
	cfg, err := config.NewProcessorConfig()
	if err != nil {
		panic(err)
	}

	return &processorTestSuite{
		cfg: cfg,
	}
}

func newDefProcessor(s *processorTestSuite) *EventProcessorImpl {
	return newProcessorWithCoreData(s, model.NewCoreData(
		10, 10, &model.TimeInterval{
			Start: time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
			End:   time.Date(0, 0, 0, 14, 0, 0, 0, time.UTC),
		}))
}

func newProcessorWithCoreData(s *processorTestSuite, coreData *model.CoreData) *EventProcessorImpl {
	return NewEventProcessor(
		&bytes.Buffer{},
		s.cfg,
		coreData,
		storage.NewInMemoryStorage[int, *model.IncomingEvent](),
		storage.NewInMemoryStorage[int, *model.RevenueStats](),
		storage.NewInMemoryStorage[string, int](),
		storage.NewInMemoryQueue[model.ClientData](nil),
	)

}

func TestProcessorSuite(t *testing.T) {
	suite.Run(t, newProcessorTestSuite())
}

func (s *processorTestSuite) getOutEvent(p *EventProcessorImpl) string {
	return p.out.(*bytes.Buffer).String()
}

func (s *processorTestSuite) TestInWorkingTime() {
	testCases := []struct {
		name      string
		happensAt time.Time
		ti        *model.TimeInterval
		in        bool
	}{
		{
			name:      "current day",
			happensAt: time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
			ti: &model.TimeInterval{
				Start: time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
				End:   time.Date(0, 0, 0, 14, 0, 0, 0, time.UTC),
			},
			in: true,
		},
		{
			name:      "current day, too early",
			happensAt: time.Date(0, 0, 0, 9, 0, 0, 0, time.UTC),
			ti: &model.TimeInterval{
				Start: time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
				End:   time.Date(0, 0, 0, 14, 0, 0, 0, time.UTC),
			},
		},
		{
			name:      "current day, too late",
			happensAt: time.Date(0, 0, 0, 15, 0, 0, 0, time.UTC),
			ti: &model.TimeInterval{
				Start: time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
				End:   time.Date(0, 0, 0, 14, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "next day",
			happensAt: time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC).
				AddDate(0, 0, 1),
			ti: &model.TimeInterval{
				Start: time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
				End:   time.Date(0, 0, 1, 14, 0, 0, 0, time.UTC),
			},
			in: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			p := newProcessorWithCoreData(s, &model.CoreData{
				WorkingTime: tc.ti,
			})

			s.Equal(tc.in, p.coreData.WorkingTime.In(tc.happensAt))
		})
	}
}

func (s *processorTestSuite) TestProcessArrives() {
	testCases := []struct {
		name          string
		prep          func(p *EventProcessorImpl)
		event         *model.IncomingEvent
		buildExpected func() string
	}{
		{
			name: "not open yet",
			event: model.NewIncomingEvent(
				time.Date(0, 0, 0, 9, 0, 0, 0, time.UTC),
				model.Arrives,
				model.NewClientArrives("client1"),
			),
			buildExpected: func() string {
				return model.NewErrorEvent(
					time.Date(0, 0, 0, 9, 0, 0, 0, time.UTC),
					errors.New(apierror.ErrNotOpenYet),
				).String(s.cfg.TimeFormat) + "\n"
			},
		},
		{
			name: "already in",
			event: model.NewIncomingEvent(
				time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
				model.Arrives,
				model.NewClientArrives("client1"),
			),
			prep: func(p *EventProcessorImpl) { p.clients.Set("client1", -1) },
			buildExpected: func() string {
				return model.NewErrorEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					errors.New(apierror.ErrYouShallNotPass),
				).String(s.cfg.TimeFormat) + "\n"
			},
		},
		{
			name: "ok",
			event: model.NewIncomingEvent(
				time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
				model.Arrives,
				model.NewClientArrives("client1"),
			),
			buildExpected: func() string { return "" },
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			p := newDefProcessor(s)

			if tc.prep != nil {
				tc.prep(p)
			}

			p.processArrives(tc.event)
			s.Equal(tc.buildExpected(), s.getOutEvent(p))

			if tc.buildExpected() == "" {
				table, ok := p.clients.Get(tc.event.Client.GetName())
				s.True(ok)
				s.Equal(-1, table)
			}
		})
	}
}

func (s *processorTestSuite) TestProcessSits() {
	testCases := []struct {
		name          string
		prep          func(p *EventProcessorImpl)
		buildEvent    func(p *EventProcessorImpl) *model.IncomingEvent
		generateSat   bool
		buildExpected func(p *EventProcessorImpl) string
		buildRevenue  func(p *EventProcessorImpl) *model.RevenueStats
		finalCheck    func(p *EventProcessorImpl)
		prevTable     int
	}{
		{
			name: "unknown client",
			buildEvent: func(p *EventProcessorImpl) *model.IncomingEvent {
				return model.NewIncomingEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.Sits,
					model.NewClientSits("client1", 1, p.coreData.TablesCount),
				)
			},
			buildExpected: func(p *EventProcessorImpl) string {
				return model.NewErrorEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					errors.New(apierror.ErrClientUnknown),
				).String(s.cfg.TimeFormat) + "\n"
			},
			buildRevenue: func(p *EventProcessorImpl) *model.RevenueStats {
				return nil
			},
		},
		{
			name: "table is busy",
			buildEvent: func(p *EventProcessorImpl) *model.IncomingEvent {
				return model.NewIncomingEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.Sits,
					model.NewClientSits("client1", 1, p.coreData.TablesCount),
				)
			},
			prep: func(p *EventProcessorImpl) {
				p.clients.Set("client1", -1)
				p.tables.Set(1, model.NewIncomingEvent(
					time.Date(0, 0, 0, 9, 0, 0, 0, time.UTC),
					model.Sits,
					model.NewClientSits("client1", 1, p.coreData.TablesCount),
				))
			},
			buildExpected: func(p *EventProcessorImpl) string {
				return model.NewErrorEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					errors.New(apierror.ErrTableIsBusy),
				).String(s.cfg.TimeFormat) + "\n"
			},
			buildRevenue: func(p *EventProcessorImpl) *model.RevenueStats {
				return nil
			},
		},
		{
			name: "ok",
			buildEvent: func(p *EventProcessorImpl) *model.IncomingEvent {
				return model.NewIncomingEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.Sits,
					model.NewClientSits("client1", 1, p.coreData.TablesCount),
				)
			},
			buildExpected: func(p *EventProcessorImpl) string { return "" },
			buildRevenue: func(p *EventProcessorImpl) *model.RevenueStats {
				return nil
			},
			prep: func(p *EventProcessorImpl) {
				p.clients.Set("client1", -1)
			},
		},
		{
			name: "ok, generate sat event",
			buildEvent: func(p *EventProcessorImpl) *model.IncomingEvent {
				return model.NewIncomingEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.Sits,
					model.NewClientSits("client1", 1, p.coreData.TablesCount),
				)
			},
			generateSat: true,
			buildExpected: func(p *EventProcessorImpl) string {
				return model.NewClientSatEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.NewClientSits("client1", 1, p.coreData.TablesCount),
				).String(s.cfg.TimeFormat) + "\n"
			},
			buildRevenue: func(p *EventProcessorImpl) *model.RevenueStats {
				return nil
			},
			prep: func(p *EventProcessorImpl) {
				p.clients.Set("client1", -1)
			},
		},
		{
			name: "ok, changing table",
			buildEvent: func(p *EventProcessorImpl) *model.IncomingEvent {
				return model.NewIncomingEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.Sits,
					model.NewClientSits("client1", 1, p.coreData.TablesCount),
				)
			},
			buildExpected: func(p *EventProcessorImpl) string { return "" },
			prep: func(p *EventProcessorImpl) {
				p.clients.Set("client1", 2)
				p.tables.Set(2, model.NewIncomingEvent(
					time.Date(0, 0, 0, 9, 0, 0, 0, time.UTC),
					model.Sits,
					model.NewClientSits("client1", 2, p.coreData.TablesCount),
				))
			},
			buildRevenue: func(p *EventProcessorImpl) *model.RevenueStats {
				return &model.RevenueStats{
					Income:    p.coreData.PricePerHour * 3,
					UsageTime: time.Duration(3) * time.Hour,
				}
			},
			finalCheck: func(p *EventProcessorImpl) {
				prevTable, ok := p.tables.Get(2)
				s.False(ok)
				s.Nil(prevTable)
			},
			prevTable: 2,
		},
		{
			name: "ok, change with lower than hour",
			buildEvent: func(p *EventProcessorImpl) *model.IncomingEvent {
				return model.NewIncomingEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.Sits,
					model.NewClientSits("client1", 1, p.coreData.TablesCount),
				)
			},
			buildExpected: func(p *EventProcessorImpl) string { return "" },
			prep: func(p *EventProcessorImpl) {
				p.clients.Set("client1", 2)
				p.tables.Set(2, model.NewIncomingEvent(
					time.Date(0, 0, 0, 11, 30, 0, 0, time.UTC),
					model.Sits,
					model.NewClientSits("client1", 2, p.coreData.TablesCount),
				))
			},
			buildRevenue: func(p *EventProcessorImpl) *model.RevenueStats {
				return &model.RevenueStats{
					Income:    p.coreData.PricePerHour * 1,
					UsageTime: time.Duration(30) * time.Minute,
				}
			},
			finalCheck: func(p *EventProcessorImpl) {
				prevTable, ok := p.tables.Get(2)
				s.False(ok)
				s.Nil(prevTable)
			},
			prevTable: 2,
		},
	}

	for _, tc := range testCases {
		_ = tc
		s.Run(tc.name, func() {
			p := newDefProcessor(s)

			if tc.prep != nil {
				tc.prep(p)
			}

			event := tc.buildEvent(p)
			expected := tc.buildExpected(p)
			p.processSits(event, tc.generateSat)
			s.Equal(expected, s.getOutEvent(p))

			if tc.finalCheck != nil {
				tc.finalCheck(p)
			}

			if tc.buildExpected(p) == "" {
				table, ok := p.tables.Get(event.Client.(*model.ClientSits).GetTable())
				if ok {
					s.Equal(event, table)
				}
			}

			revenue, ok := p.revenue.Get(tc.prevTable)
			expRevenue := tc.buildRevenue(p)

			if ok {
				s.Equal(expRevenue, revenue)
			} else {
				s.Nil(expRevenue)
			}
		})
	}
}

func (s *processorTestSuite) TestProcessWaits() {
	testCases := []struct {
		name          string
		buildEvent    func(p *EventProcessorImpl) *model.IncomingEvent
		buildExpected func(p *EventProcessorImpl) string
		prep          func(p *EventProcessorImpl)
		check         func(p *EventProcessorImpl)
	}{
		{
			name: "have free tables",
			buildEvent: func(p *EventProcessorImpl) *model.IncomingEvent {
				return model.NewIncomingEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.Waits,
					model.NewClientWaits("client1"),
				)
			},
			buildExpected: func(p *EventProcessorImpl) string {
				return model.NewErrorEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					errors.New(apierror.ErrCantWaitLonger),
				).String(p.cfg.TimeFormat) + "\n"
			},
		},
		{
			name: "queue is full",
			buildEvent: func(p *EventProcessorImpl) *model.IncomingEvent {
				return model.NewIncomingEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.Waits,
					model.NewClientWaits("client1"),
				)
			},
			buildExpected: func(p *EventProcessorImpl) string {
				return model.NewClientLeftEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.NewClientWaits("client1"),
				).String(p.cfg.TimeFormat) + "\n"
			},
			prep: func(p *EventProcessorImpl) {
				p.coreData.TablesCount = 1
				p.tables.Set(1, model.NewIncomingEvent(
					time.Date(0, 0, 0, 11, 30, 0, 0, time.UTC),
					model.Sits,
					model.NewClientSits("client2", 1, p.coreData.TablesCount),
				))

				p.waitingQueue.Push(model.NewClientWaits("client1"))
			},
		},
		{
			name: "have free tables and queue is not full",
			buildEvent: func(p *EventProcessorImpl) *model.IncomingEvent {
				return model.NewIncomingEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.Waits,
					model.NewClientWaits("client1"),
				)
			},
			buildExpected: func(p *EventProcessorImpl) string {
				return ""
			},
			prep: func(p *EventProcessorImpl) {
				p.coreData.TablesCount = 1
				p.tables.Set(1, model.NewIncomingEvent(
					time.Date(0, 0, 0, 11, 30, 0, 0, time.UTC),
					model.Sits,
					model.NewClientSits("client2", 1, p.coreData.TablesCount),
				))
			},
			check: func(p *EventProcessorImpl) {
				s.Equal(1, p.waitingQueue.Len())
				top, err := p.waitingQueue.Peek()
				s.NoError(err)
				s.Equal(model.NewClientWaits("client1"), top)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			p := newDefProcessor(s)

			if tc.prep != nil {
				tc.prep(p)
			}

			event := tc.buildEvent(p)
			p.processWaits(event)

			expected := tc.buildExpected(p)
			s.Equal(expected, s.getOutEvent(p))

			if tc.check != nil {
				tc.check(p)
			}
		})
	}
}

func (s *processorTestSuite) TestProcessLeaves() {
	testCases := []struct {
		name          string
		buildEvent    func(p *EventProcessorImpl) *model.IncomingEvent
		buildExpected func(p *EventProcessorImpl) string
		prep          func(p *EventProcessorImpl)
		check         func(p *EventProcessorImpl)
		buildRevenue  func(p *EventProcessorImpl) *model.RevenueStats
		generateLeft  bool
		prevTable     int
	}{
		{
			name: "unknown client",
			buildEvent: func(p *EventProcessorImpl) *model.IncomingEvent {
				return model.NewIncomingEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.Leaves,
					model.NewClientLeaves("client1"),
				)
			},
			buildExpected: func(p *EventProcessorImpl) string {
				return model.NewErrorEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					errors.New(apierror.ErrClientUnknown),
				).String(p.cfg.TimeFormat) + "\n"
			},
		},
		{
			name: "client only arrived",
			buildEvent: func(p *EventProcessorImpl) *model.IncomingEvent {
				return model.NewIncomingEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.Leaves,
					model.NewClientLeaves("client1"),
				)
			},
			buildExpected: func(p *EventProcessorImpl) string {
				return model.NewClientLeftEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.NewClientLeaves("client1"),
				).String(p.cfg.TimeFormat) + "\n"
			},
			prep: func(p *EventProcessorImpl) {
				p.clients.Set("client1", -1)
			},
			generateLeft: true,
			check: func(p *EventProcessorImpl) {
				s.Equal(0, p.clients.Len())
				s.Equal(0, p.tables.Len())
			},
		},
		{
			name: "client only arrived, no gen",
			buildEvent: func(p *EventProcessorImpl) *model.IncomingEvent {
				return model.NewIncomingEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.Leaves,
					model.NewClientLeaves("client1"),
				)
			},
			buildExpected: func(p *EventProcessorImpl) string {
				return ""
			},
			prep: func(p *EventProcessorImpl) {
				p.clients.Set("client1", -1)
			},
			check: func(p *EventProcessorImpl) {
				s.Equal(0, p.clients.Len())
				s.Equal(0, p.tables.Len())
			},
		},
		{
			name: "client sits and leaves",
			buildEvent: func(p *EventProcessorImpl) *model.IncomingEvent {
				return model.NewIncomingEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.Leaves,
					model.NewClientLeaves("client1"),
				)
			},
			buildExpected: func(p *EventProcessorImpl) string {
				return model.NewClientLeftEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.NewClientLeaves("client1"),
				).String(p.cfg.TimeFormat) + "\n"
			},
			prep: func(p *EventProcessorImpl) {
				p.clients.Set("client1", 1)
				p.tables.Set(1, model.NewIncomingEvent(
					time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
					model.Sits,
					model.NewClientSits("client1", 1, p.coreData.TablesCount),
				))
			},
			generateLeft: true,
			check: func(p *EventProcessorImpl) {
				s.Equal(0, p.clients.Len())
				s.Equal(0, p.tables.Len())
			},
			buildRevenue: func(p *EventProcessorImpl) *model.RevenueStats {
				return &model.RevenueStats{
					Income:    p.coreData.PricePerHour * 2,
					UsageTime: time.Duration(2) * time.Hour,
				}
			},
			prevTable: 1,
		},
		{
			name: "client sits and leaves, popped from queue",
			buildEvent: func(p *EventProcessorImpl) *model.IncomingEvent {
				return model.NewIncomingEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.Leaves,
					model.NewClientLeaves("client1"),
				)
			},
			buildExpected: func(p *EventProcessorImpl) string {
				leftEvent := model.NewClientLeftEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.NewClientLeaves("client1"),
				).String(p.cfg.TimeFormat) + "\n"

				satEvent := model.NewClientSatEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.NewClientSits("client2", 1, p.coreData.TablesCount),
				).String(p.cfg.TimeFormat) + "\n"

				return leftEvent + satEvent
			},
			prep: func(p *EventProcessorImpl) {
				p.clients.Set("client1", 1)
				p.tables.Set(1, model.NewIncomingEvent(
					time.Date(0, 0, 0, 10, 59, 0, 0, time.UTC),
					model.Sits,
					model.NewClientSits("client1", 1, p.coreData.TablesCount),
				))
				p.waitingQueue.Push(model.NewClientWaits("client2"))
			},
			generateLeft: true,
			check: func(p *EventProcessorImpl) {
				s.Equal(1, p.clients.Len())
				s.Equal(1, p.tables.Len())
				s.Equal(0, p.waitingQueue.Len())

				_, ok := p.clients.Get("client2")
				s.True(ok)

				_, ok = p.tables.Get(1)
				s.True(ok)
			},
			buildRevenue: func(p *EventProcessorImpl) *model.RevenueStats {
				return &model.RevenueStats{
					Income:    p.coreData.PricePerHour * 2,
					UsageTime: time.Duration(1)*time.Hour + time.Duration(1)*time.Minute,
				}
			},
			prevTable: 1,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			p := newDefProcessor(s)

			if tc.prep != nil {
				tc.prep(p)
			}

			event := tc.buildEvent(p)
			p.processLeaves(event, tc.generateLeft)

			expected := tc.buildExpected(p)
			s.Equal(expected, s.getOutEvent(p))

			if tc.check != nil {
				tc.check(p)
			}

			if tc.buildRevenue != nil {
				revenue, ok := p.revenue.Get(tc.prevTable)
				expRevenue := tc.buildRevenue(p)

				if ok {
					s.Equal(expRevenue, revenue)
				} else {
					s.Nil(expRevenue)
				}
			}
		})
	}
}

func (s *processorTestSuite) TestProcessRevenueUpdate() {
	testCases := []struct {
		name         string
		tables       []int
		happensAt    []time.Time
		buildRevenue func(p *EventProcessorImpl) map[int]*model.RevenueStats
		prep         func(p *EventProcessorImpl)
	}{
		{
			name:   "single table",
			tables: []int{1},
			happensAt: []time.Time{
				time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
			},
			buildRevenue: func(p *EventProcessorImpl) map[int]*model.RevenueStats {
				return map[int]*model.RevenueStats{
					1: {
						Income:    p.coreData.PricePerHour * 2,
						UsageTime: time.Duration(2) * time.Hour,
					},
				}
			},
			prep: func(p *EventProcessorImpl) {
				p.tables.Set(1, model.NewIncomingEvent(
					time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
					model.Sits,
					model.NewClientSits("client1", 1, p.coreData.TablesCount),
				))
			},
		},
		{
			name:   "no tables",
			tables: []int{-1},
			happensAt: []time.Time{
				time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
			},
			buildRevenue: func(p *EventProcessorImpl) map[int]*model.RevenueStats {
				return map[int]*model.RevenueStats{}
			},
		},
		{
			name:   "multiple tables",
			tables: []int{1, 2, 3},
			happensAt: []time.Time{
				time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
				time.Date(0, 0, 0, 13, 0, 0, 0, time.UTC),
				time.Date(0, 0, 0, 14, 0, 0, 0, time.UTC),
			},
			buildRevenue: func(p *EventProcessorImpl) map[int]*model.RevenueStats {
				return map[int]*model.RevenueStats{
					1: {
						Income:    p.coreData.PricePerHour * 2,
						UsageTime: time.Duration(2) * time.Hour,
					},
					2: {
						Income:    p.coreData.PricePerHour * 3,
						UsageTime: time.Duration(3) * time.Hour,
					},
					3: {
						Income:    p.coreData.PricePerHour * 4,
						UsageTime: time.Duration(3)*time.Hour + time.Duration(49)*time.Minute,
					},
				}
			},
			prep: func(p *EventProcessorImpl) {
				p.tables.Set(1, model.NewIncomingEvent(
					time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
					model.Sits,
					model.NewClientSits("client1", 1, p.coreData.TablesCount),
				))
				p.tables.Set(2, model.NewIncomingEvent(
					time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
					model.Sits,
					model.NewClientSits("client2", 2, p.coreData.TablesCount),
				))
				p.tables.Set(3, model.NewIncomingEvent(
					time.Date(0, 0, 0, 10, 11, 0, 0, time.UTC),
					model.Sits,
					model.NewClientSits("client3", 3, p.coreData.TablesCount),
				))
			},
		},
		{
			name:   "single table multiple events",
			tables: []int{1, 1},
			happensAt: []time.Time{
				time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
				time.Date(0, 0, 0, 13, 0, 0, 0, time.UTC),
			},
			buildRevenue: func(p *EventProcessorImpl) map[int]*model.RevenueStats {
				return map[int]*model.RevenueStats{
					1: {
						Income:    p.coreData.PricePerHour*2 + p.coreData.PricePerHour*3,
						UsageTime: time.Duration(2)*time.Hour + time.Duration(3)*time.Hour,
					},
				}
			},
			prep: func(p *EventProcessorImpl) {
				p.tables.Set(1, model.NewIncomingEvent(
					time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
					model.Sits,
					model.NewClientSits("client1", 1, p.coreData.TablesCount),
				))
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			p := newDefProcessor(s)

			if tc.prep != nil {
				tc.prep(p)
			}

			for i, table := range tc.tables {
				p.updateRevenue(table, tc.happensAt[i])
			}

			for table, expRevenue := range tc.buildRevenue(p) {
				revenue, ok := p.revenue.Get(table)
				if ok {
					s.Equal(expRevenue, revenue)
				} else {
					s.Equal(expRevenue, 0)
				}
			}
		})
	}
}

func (s *processorTestSuite) TestProcessEvents() {
	testCases := []struct {
		name          string
		events        []*model.IncomingEvent
		parsingErr    error
		expErr        error
		buildExpected func(p *EventProcessorImpl) string
	}{
		{
			name: "single event",
			events: []*model.IncomingEvent{
				model.NewIncomingEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.Sits,
					model.NewClientSits("client1", 1, 1),
				),
			},
			buildExpected: func(p *EventProcessorImpl) string {
				startTime := p.coreData.WorkingTime.Start.Format(p.cfg.TimeFormat)
				endTime := p.coreData.WorkingTime.End.Format(p.cfg.TimeFormat)

				inSitEvent := model.NewIncomingEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					model.Sits,
					model.NewClientSits("client1", 1, 1),
				).String(s.cfg.TimeFormat)

				outSitEvent := model.NewErrorEvent(
					time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC),
					errors.New(apierror.ErrClientUnknown),
				).String(s.cfg.TimeFormat)

				return strings.Join([]string{
					startTime,
					inSitEvent,
					outSitEvent,
					endTime,
				}, "\n") + "\n"
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			p := newDefProcessor(s)

			events := make(chan model.WrappedIncomingEvent)
			go func() {
				defer close(events)

				for _, event := range tc.events {
					events <- model.WrappedIncomingEvent{Event: event}
				}

				if tc.parsingErr != nil {
					events <- model.WrappedIncomingEvent{Err: tc.parsingErr}
				}
			}()

			done := make(chan error)
			defer close(done)

			go func() {
				e := p.ProcessEvents(events)
				if e != nil {
					done <- e
				} else {
					p.ShowRevenue()
					done <- nil
				}
			}()

			s.Equal(tc.expErr, <-done)

			outEvent := s.getOutEvent(p)
			expOutEvent := tc.buildExpected(p)
			s.Equal(outEvent, expOutEvent)
		})
	}
}
