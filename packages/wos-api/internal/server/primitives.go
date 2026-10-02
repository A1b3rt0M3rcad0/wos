package server

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

type uuidV7Generator struct{}

func (uuidV7Generator) NewID() (domain.ID, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", domain.WrapError(domain.ErrorCodeInvalidArgument, "generate UUIDv7 randomness", err)
	}

	millis := uint64(time.Now().UTC().UnixMilli())
	raw[0] = byte(millis >> 40)
	raw[1] = byte(millis >> 32)
	raw[2] = byte(millis >> 24)
	raw[3] = byte(millis >> 16)
	raw[4] = byte(millis >> 8)
	raw[5] = byte(millis)
	raw[6] = (raw[6] & 0x0f) | 0x70
	raw[8] = (raw[8] & 0x3f) | 0x80

	value := fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(raw[0:4]),
		binary.BigEndian.Uint16(raw[4:6]),
		binary.BigEndian.Uint16(raw[6:8]),
		binary.BigEndian.Uint16(raw[8:10]),
		uint64(raw[10])<<40|uint64(raw[11])<<32|uint64(raw[12])<<24|
			uint64(raw[13])<<16|uint64(raw[14])<<8|uint64(raw[15]),
	)
	return domain.ParseID(value)
}
