package apierror

const (

	// ErrYouShallNotPass is generated when the client is already in the computer club.
	ErrYouShallNotPass = "YouShallNotPass"

	// ErrNotOpenYet is generated when the client tries to enter the computer club before it opens.
	ErrNotOpenYet = "NotOpenYet"

	// ErrClientUnknown is generated when the client is not in the computer club.
	ErrClientUnknown = "ClientUnknown"

	// ErrTableIsBusy is generated when the client tries to sit on the table that is already occupied.
	ErrTableIsBusy = "PlaceIsBusy"

	// ErrCantWaitLonger is generated when the client tries to wait for a table, but some tables are free.
	ErrCantWaitLonger = "ICanWaitNoLonger!"
)
