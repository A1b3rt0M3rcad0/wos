// Verify old producer facts with the current codec without opening/writing a DB.
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"io"
	"os"
)

func main() {
	if err := verify(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func verify() error {
	var input struct {
		Keys  []d.SigningKey `json:"keys"`
		Facts []d.SignedFact `json:"facts"`
	}
	decoder := json.NewDecoder(io.LimitReader(os.Stdin, 32<<20))
	if err := decoder.Decode(&input); err != nil {
		return err
	}
	keys := map[d.ID]d.SigningKey{}
	for _, key := range input.Keys {
		keys[key.ID] = key
	}
	counts := map[string]int{}
	for _, fact := range input.Facts {
		if err := fact.Validate(); err != nil {
			return err
		}
		key, ok := keys[fact.KeyID]
		if !ok {
			return fmt.Errorf("historical public signing key missing")
		}
		public, err := key.PublicBytes()
		if err != nil {
			return err
		}
		var proof signing.Proof
		if err = json.Unmarshal(fact.Proof, &proof); err != nil {
			return err
		}
		purpose := signing.ContractSpec
		switch fact.Kind {
		case "authority":
			purpose = signing.ContractAuthority
		case "acceptance":
			purpose = signing.AcceptanceReceipt
		case "return":
			purpose = signing.WorkReturn
			if fact.ContractKind == "review" {
				purpose = signing.ReviewReturn
			}
		}
		envelope := signing.Envelope{PayloadType: proof.PayloadType, Payload: base64.StdEncoding.EncodeToString(fact.Payload), Signatures: []signing.Signature{{KeyID: proof.KeyID, Sig: proof.Signature}}}
		verified, err := signing.Verify(envelope, purpose, fact.KeyID.String(), public)
		if err != nil {
			return err
		}
		if signing.Digest(verified) != fact.PayloadDigest {
			return fmt.Errorf("historical exact payload digest changed")
		}
		counts[fact.Kind]++
	}
	for _, kind := range []string{"specification", "authority", "return", "acceptance"} {
		if counts[kind] == 0 {
			return fmt.Errorf("actual historical fixture lacks %s facts", kind)
		}
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"verified": true, "facts": len(input.Facts), "kinds": counts, "historical_public_keys_only": true, "re_signed": false})
}
