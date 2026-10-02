package sqlite

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func encodeTime(value time.Time) int64 {
	return value.UTC().UnixMicro()
}

func decodeTime(value int64) time.Time {
	return time.UnixMicro(value).UTC()
}

func encodeOptionalTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return encodeTime(*value)
}

func marshalJSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func unmarshalJSON(value string, target any) error {
	if err := json.Unmarshal([]byte(value), target); err != nil {
		return err
	}
	return nil
}

func conclusionStorageID(owner domain.EntityRef, ordinal int, conclusion domain.Conclusion) (string, error) {
	actorJSON, err := marshalJSON(conclusion.Actor)
	if err != nil {
		return "", err
	}
	assessmentsJSON, err := marshalJSON(conclusion.Assessments)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf(
		"%s\x00%s\x00%s\x00%d\x00%s\x00%s\x00%s\x00%d\x00%s",
		owner.NamespaceID,
		owner.OutcomeID,
		owner.ID,
		ordinal,
		conclusion.PrincipalID,
		actorJSON,
		conclusion.Reason,
		conclusion.ConcludedAt.UTC().UnixMicro(),
		assessmentsJSON,
	)))
	return hex.EncodeToString(sum[:]), nil
}
