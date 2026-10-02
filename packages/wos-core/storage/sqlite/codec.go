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

func legacyConclusionPublicID(storageID string, recordedAt time.Time) (domain.ID, error) {
	sum := sha256.Sum256([]byte(storageID))
	ms := uint64(recordedAt.UTC().UnixMilli())

	var raw [16]byte
	raw[0] = byte(ms >> 40)
	raw[1] = byte(ms >> 32)
	raw[2] = byte(ms >> 24)
	raw[3] = byte(ms >> 16)
	raw[4] = byte(ms >> 8)
	raw[5] = byte(ms)
	raw[6] = 0x70 | (sum[0] & 0x0f)
	raw[7] = sum[1]
	raw[8] = 0x80 | (sum[2] & 0x3f)
	copy(raw[9:], sum[3:10])

	value := fmt.Sprintf(
		"%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		raw[0], raw[1], raw[2], raw[3],
		raw[4], raw[5],
		raw[6], raw[7],
		raw[8], raw[9],
		raw[10], raw[11], raw[12], raw[13], raw[14], raw[15],
	)
	return domain.ParseID(value)
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
