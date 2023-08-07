package apierror

const (
	ErrTablesCountNotSpecified  = "tables count are not specified"
	ErrTablesCountInvalidFormat = "tables count are not integer"

	ErrPricePerHourNotSpecified  = "price per hour are not specified"
	ErrPricePerHourInvalidFormat = "price per hour are not integer"

	ErrWorkingTimeNotSpecified  = "working time are not specified"
	ErrWorkingTimeInvalidFormat = "working time are not time interval"
	ErrFailedToParseStartTime   = "failed to parse start time"
	ErrFailedToParseEndTime     = "failed to parse end time"

	ErrEventInvalidFormat             = "event must be in format: <time> <event-type> <client-data>"
	ErrFailedToParseEventTime         = "failed to parse event happened time"
	ErrFailedToParseEventType         = "failed to parse event type"
	ErrUnknownEventType               = "unknown event type"
	ErrClientDataInvalidFormat        = "invalid client data format for event type"
	ErrClientDataInvalidName          = "invalid client name"
	ErrFailedToParseClientTableNumber = "failed to parse client table number"

	ErrValueMustBeMoreThanZero = "value must be more than zero"
	ErrValueTooBig             = "value is too big"
)
