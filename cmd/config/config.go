package config

import "github.com/ilyakaznacheev/cleanenv"

type Parser struct {

	// TimeFormat is a format for time.Time.Format() function
	// See https://golang.org/pkg/time/#Time.Format
	//
	// Example: "15:04" for taking only hours and minutes
	TimeFormat string `env:"TIME_FORMAT" env-default:"15:04"`

	// TimeSeparator is a separator between start and end time
	//
	// Example: " " for "10:00 18:00"
	TimeSeparator string `env:"TIME_SEPARATOR" env-default:" "`

	// EventInfoSeparator is a separator between events info
	//
	// Example: " " for "10:00 1 John"
	EventInfoSeparator string `env:"EVENT_INFO_SEPARATOR" env-default:" "`

	// DistinctEventInfoCount is a count of distinct event info
	//
	// Example: 3 for "10:00 1 John", for other event types can be more but always >= 3
	DistinctEventInfoCount int `env:"DISTINCT_EVENT_INFO_COUNT" env-default:"3"`

	// EventsChanSize is a size of events channel
	EventsChanSize int `env:"EVENTS_CHAN_SIZE" env-default:"10"`
}

type Processor struct {

	// TimeFormat is a format for time.Time.Format() function
	// See https://golang.org/pkg/time/#Time.Format
	//
	// Example: "15:04" for taking only hours and minutes
	TimeFormat string `env:"TIME_FORMAT" env-default:"15:04"`
}

func NewParserConfig() (*Parser, error) {
	var cfg Parser
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func NewProcessorConfig() (*Processor, error) {
	var cfg Processor
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
