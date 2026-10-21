package domain

// OperationalProfile versions the business rules applied to an Outcome. It is
// independent from the Namespace's work-authority protocol.
type OperationalProfile string

const (
	OperationalProfileCompatibilityV1 OperationalProfile = "compatibility_v1"
	OperationalProfileOperationalV2   OperationalProfile = "operational_v2"
)

// Effective returns the compatibility behavior for records created before
// operational profiles were persisted.
func (p OperationalProfile) Effective() OperationalProfile {
	if p == "" {
		return OperationalProfileCompatibilityV1
	}
	return p
}

func (p OperationalProfile) Valid() bool {
	switch p.Effective() {
	case OperationalProfileCompatibilityV1, OperationalProfileOperationalV2:
		return true
	default:
		return false
	}
}
