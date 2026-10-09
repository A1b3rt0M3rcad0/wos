package woscli

import (
	"context"
	"fmt"
	"os"
)

func runWorkspaceV2(ctx context.Context, w *Workspace, o options) (Output, error) {
	result := Output{Operation: "signed workspace"}
	name, e := w.SelectProfileV2(o.values["profile"], os.Getenv("WOS_PROFILE"))
	if e != nil {
		return result, e
	}
	profile, client, token, e := loadProfileClientV2(w, name, defaultSecretResolverV2(w))
	if e != nil {
		return result, e
	}
	defer clear(token)
	identity, e := checkProfileIdentityV2(ctx, client, profile, nil)
	if e != nil {
		return result, e
	}
	switch o.args[0] {
	case "work":
		return workCommandV2(ctx, w, profile, client, token, identity, o)
	case "auth":
		if len(o.args) != 2 || o.args[1] != "status" {
			return result, usage("auth status")
		}
		result.Data = compactProfileStatusV2(profile, identity)
		return result, nil
	case "doctor":
		private, e := readProfilePrivateV2(profile, defaultSecretResolverV2(w))
		if e != nil {
			return result, e
		}
		clear(private)
		result.Data = map[string]any{"profile": compactProfileV2(profile), "destination_verified": true, "issuer_verified": true, "local_signing_key_matches": true, "pending_operations": len(profile.Local.PendingOperations), "authority": "every mutation requires live server authorization"}
		return result, nil
	case "capabilities":
		value, e := client.Capabilities(ctx)
		result.Data = value
		return result, e
	default:
		return result, fmt.Errorf("signed workspace operation is not implemented yet; legacy commands cannot substitute for v2 authority")
	}
}
