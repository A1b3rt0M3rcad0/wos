package domain

import "math"

// Version is the optimistic-lock version of a mutable aggregate.
type Version uint64

const InitialVersion Version = 1

func (v Version) Validate() error {
	if v < InitialVersion {
		return NewError(ErrorCodeInvalidVersion, "aggregate version must be at least 1")
	}
	return nil
}

func (v Version) Next() (Version, error) {
	if err := v.Validate(); err != nil {
		return 0, err
	}
	if uint64(v) == math.MaxUint64 {
		return 0, NewError(ErrorCodeInvalidVersion, "aggregate version overflow")
	}
	return v + 1, nil
}

// OutcomeRevision is the monotonically increasing transactional revision of
// all confirmed domain changes inside one Outcome. Zero is the initial state of
// a newly created coordination row before its first confirmed command.
type OutcomeRevision uint64

const InitialOutcomeRevision OutcomeRevision = 0

func (r OutcomeRevision) Next() (OutcomeRevision, error) {
	if uint64(r) == math.MaxUint64 {
		return 0, NewError(ErrorCodeInvalidVersion, "outcome revision overflow")
	}
	return r + 1, nil
}
