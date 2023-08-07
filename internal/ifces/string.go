package ifces

type TimeFormatter interface {
	String(timeFormat string) string
}
