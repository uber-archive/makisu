package utils

// BuildHash is a variable that will be populated at build-time of the
// binary via the ldflags parameter. It is used to break cache when a new
// version of makisu is used.
var BuildHash string

// We need an init function for now to go around the github issue listed above.
func init() {
	if BuildHash == "" {
		BuildHash = "build-hash-2018-3-28"
	}
}
