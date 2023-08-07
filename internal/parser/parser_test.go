package parser

import (
	"bufio"
	"github.com/stretchr/testify/suite"
	"log"
	"strings"
	"testing"
	"time"
	"yadro-intern/cmd/config"
	"yadro-intern/internal/apierror"
	"yadro-intern/internal/model"
)

type parserSuite struct {
	suite.Suite
	cfg *config.Parser
}

func newParserSuite() *parserSuite {
	cfg, err := config.NewParserConfig()
	if err != nil {
		log.Fatal(err)
	}

	return &parserSuite{
		cfg: cfg,
	}
}

func TestParserSuite(t *testing.T) {
	suite.Run(t, newParserSuite())
}

func scannerFromStr(s string) *bufio.Scanner {
	return bufio.NewScanner(strings.NewReader(s))
}

func (s *parserSuite) compareHM(t1, t2 time.Time) {
	s.Equal(t1.Hour(), t2.Hour())
	s.Equal(t1.Minute(), t2.Minute())
}

func (s *parserSuite) compareTimeIntervals(t1, t2 *model.TimeInterval) {
	if t1 == nil || t2 == nil {
		s.Equal(t1, t2)
		return
	}

	s.compareHM(t1.Start, t2.Start)
	s.compareHM(t1.End, t2.End)
}

func (s *parserSuite) compareErrors(e1, e2 error) {
	if e1 == nil || e2 == nil {
		s.Equal(e1, e2)
		return
	}

	s.Equal(e1.Error(), e2.Error())
}

func (s *parserSuite) compareCoreData(c1, c2 *model.CoreData) {
	if c1 == nil || c2 == nil {
		s.Equal(c1, c2)
		return
	}

	s.Equal(c1.TablesCount, c2.TablesCount)
	s.Equal(c1.PricePerHour, c2.PricePerHour)
	s.compareTimeIntervals(c1.WorkingTime, c2.WorkingTime)
}

func (s *parserSuite) compareEvent(e1, e2 *model.IncomingEvent) {
	if e1 == nil || e2 == nil {
		s.Equal(e1, e2)
		return
	}

	s.Equal(e1.Type, e2.Type)
	s.compareHM(e1.HappensAt, e2.HappensAt)
	s.compareClientData(e1.Client, e2.Client)
}

func (s *parserSuite) compareClientData(d1, d2 model.ClientData) {
	s.Equal(d1.GetName(), d2.GetName())

	sits1, ok := d1.(*model.ClientSits)
	sits2, ok2 := d2.(*model.ClientSits)

	if ok && ok2 {
		s.Equal(sits1.GetTable(), sits2.GetTable())
	}

	if !ok && !ok2 {
		return
	}

	s.Equal(ok, ok2, "one of the clients is ClientSits, another is not")
}

func (s *parserSuite) TestParser_ReadTablesCount() {
	testCases := []struct {
		name   string
		input  string
		exp    int
		expErr error
	}{
		{
			name:  "valid tables count",
			input: "10",
			exp:   10,
		},
		{
			name:   "invalid tables count",
			input:  "10.5",
			expErr: &apierror.ParseError{RowNumber: 1, UserMsg: apierror.ErrTablesCountInvalidFormat},
		},
		{
			name:   "empty tables count",
			input:  "",
			expErr: &apierror.ParseError{RowNumber: 1, UserMsg: apierror.ErrTablesCountNotSpecified},
		},
		{
			name:   "negative tables count",
			input:  "-10",
			expErr: &apierror.ValidationError{RowNumber: 1, UserMsg: apierror.ErrValueMustBeMoreThanZero},
		},
		{
			name:   "zero tables count",
			input:  "0",
			expErr: &apierror.ValidationError{RowNumber: 1, UserMsg: apierror.ErrValueMustBeMoreThanZero},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			p := NewFileParser(scannerFromStr(tc.input), s.cfg)

			got, err := p.readTablesCount(apierror.MoreThenZero)
			s.compareErrors(tc.expErr, err)
			s.Equal(tc.exp, got)
		})
	}
}

func (s *parserSuite) TestParser_ReadWorkingTime() {
	testCases := []struct {
		name   string
		input  string
		exp    *model.TimeInterval
		expErr error
	}{
		{
			name:  "valid working time",
			input: "10:00 20:00",
			exp: model.NewTimeInterval(
				time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
				time.Date(0, 0, 0, 20, 0, 0, 0, time.UTC),
			),
		},
		{
			name:   "invalid working time format",
			input:  "10:00 20:00 23:00",
			expErr: &apierror.ParseError{RowNumber: 1, UserMsg: apierror.ErrWorkingTimeInvalidFormat},
		},
		{
			name:   "start time parse error",
			input:  "ab:00 20:00",
			expErr: &apierror.ParseError{RowNumber: 1, UserMsg: apierror.ErrFailedToParseStartTime},
		},
		{
			name:   "end time parse error",
			input:  "10:00 ab:00",
			expErr: &apierror.ParseError{RowNumber: 1, UserMsg: apierror.ErrFailedToParseEndTime},
		},
		{
			name:  "next day working time",
			input: "20:00 10:00",
			exp: model.NewTimeInterval(
				time.Date(0, 0, 0, 20, 0, 0, 0, time.UTC),
				time.Date(0, 0, 1, 10, 0, 0, 0, time.UTC),
			),
		},
		{
			name:   "empty working time",
			input:  "",
			expErr: &apierror.ParseError{RowNumber: 1, UserMsg: apierror.ErrWorkingTimeNotSpecified},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			p := NewFileParser(scannerFromStr(tc.input), s.cfg)

			got, err := p.readWorkingTime()
			s.compareErrors(tc.expErr, err)
			s.compareTimeIntervals(tc.exp, got)
		})
	}
}

func (s *parserSuite) TestParser_ReadPricePerHour() {
	testCases := []struct {
		name   string
		input  string
		exp    int
		expErr error
	}{
		{
			name:  "valid price per hour",
			input: "10",
			exp:   10,
		},
		{
			name:   "invalid price per hour",
			input:  "10.5",
			expErr: &apierror.ParseError{RowNumber: 1, UserMsg: apierror.ErrPricePerHourInvalidFormat},
		},
		{
			name:   "empty price per hour",
			input:  "",
			expErr: &apierror.ParseError{RowNumber: 1, UserMsg: apierror.ErrPricePerHourNotSpecified},
		},
		{
			name:   "negative price per hour",
			input:  "-10",
			expErr: &apierror.ValidationError{RowNumber: 1, UserMsg: apierror.ErrValueMustBeMoreThanZero},
		},
		{
			name:   "zero price per hour",
			input:  "0",
			expErr: &apierror.ValidationError{RowNumber: 1, UserMsg: apierror.ErrValueMustBeMoreThanZero},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			p := NewFileParser(scannerFromStr(tc.input), s.cfg)

			got, err := p.readPricePerHour(apierror.MoreThenZero)
			s.compareErrors(tc.expErr, err)
			s.Equal(tc.exp, got)
		})
	}
}

func (s *parserSuite) TestParser_ReadCoreData() {
	testCases := []struct {
		name   string
		input  string
		exp    *model.CoreData
		expErr error
	}{
		{
			name:  "valid core data",
			input: "10\n10:00 20:00\n10",
			exp: &model.CoreData{
				TablesCount: 10,
				WorkingTime: model.NewTimeInterval(
					time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
					time.Date(0, 0, 0, 20, 0, 0, 0, time.UTC),
				),
				PricePerHour: 10,
			},
		},
		{
			name:   "invalid tables count",
			input:  "ab\n10:00 20:00\n10",
			expErr: &apierror.ParseError{RowNumber: 1, UserMsg: apierror.ErrTablesCountInvalidFormat},
		},
		{
			name:   "invalid working time format",
			input:  "10\n10:00 20:00 23:00\n10",
			expErr: &apierror.ParseError{RowNumber: 2, UserMsg: apierror.ErrWorkingTimeInvalidFormat},
		},
		{
			name:   "invalid price per hour",
			input:  "10\n10:00 20:00\n10.5",
			expErr: &apierror.ParseError{RowNumber: 3, UserMsg: apierror.ErrPricePerHourInvalidFormat},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			p := NewFileParser(scannerFromStr(tc.input), s.cfg)

			got, err := p.ReadCoreData()
			s.compareErrors(tc.expErr, err)
			s.compareCoreData(tc.exp, got)
		})
	}
}

func (s *parserSuite) TestParser_ReadEvent() {
	testCases := []struct {
		name   string
		input  string
		exp    *model.IncomingEvent
		expErr error
	}{
		{
			name:  "valid arrive event",
			input: "10:00 1 client1",
			exp: model.NewIncomingEvent(
				time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
				model.Arrives,
				model.NewClientArrives("client1"),
			),
		},
		{
			name:  "valid sits event",
			input: "10:00 2 client1 2",
			exp: model.NewIncomingEvent(
				time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
				model.Sits,
				model.NewClientSits("client1", 2, 3),
			),
		},
		{
			name:  "valid waits event",
			input: "10:00 3 client1",
			exp: model.NewIncomingEvent(
				time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
				model.Waits,
				model.NewClientWaits("client1"),
			),
		},
		{
			name:  "valid leaves event",
			input: "10:00 4 client1",
			exp: model.NewIncomingEvent(
				time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
				model.Leaves,
				model.NewClientLeaves("client1"),
			),
		},
		{
			name:   "invalid event type",
			input:  "10:00 5 client1",
			expErr: &apierror.ValidationError{RowNumber: 1, UserMsg: apierror.ErrUnknownEventType},
		},
		{
			name:   "invalid time format",
			input:  "10:00:00 1 client1",
			expErr: &apierror.ParseError{RowNumber: 1, UserMsg: apierror.ErrFailedToParseEventTime},
		},
		{
			name:   "invalid client name",
			input:  "10:00 1 clie@nt1",
			expErr: &apierror.ValidationError{RowNumber: 1, UserMsg: apierror.ErrClientDataInvalidName},
		},
		{
			name:   "invalid table number",
			input:  "10:00 2 client1 0",
			expErr: &apierror.ValidationError{RowNumber: 1, UserMsg: apierror.ErrValueMustBeMoreThanZero},
		},
		{
			name:   "invalid event type format",
			input:  "10:00 ab client1",
			expErr: &apierror.ParseError{RowNumber: 1, UserMsg: apierror.ErrFailedToParseEventType},
		},
		{
			name:   "table number too big",
			input:  "10:00 2 client1 11",
			expErr: &apierror.ValidationError{RowNumber: 1, UserMsg: apierror.ErrValueTooBig},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			p := NewFileParser(scannerFromStr(tc.input), s.cfg)
			p.maxTables = 3
			if !p.scanWithRowNumber() {
				s.FailNow("scanWithRowNumber() must be true")
			}

			got, err := p.readEvent()
			s.compareErrors(tc.expErr, err)
			s.compareEvent(tc.exp, got)
		})
	}
}
