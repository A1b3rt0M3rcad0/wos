package woscli

import (
	"golang.org/x/sys/windows"
	"testing"
)

// chmod does not establish a restricted Windows DACL. Fixtures must provision
// the same protected secret ACL expected from an operator-mounted secret.
func protectSecretFixtureV2(t *testing.T, path string) {
	t.Helper()
	user, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		t.Fatal(e)
	}
	descriptor, e := windows.SecurityDescriptorFromString("D:P(A;;FA;;;" + user.User.Sid.String() + ")(A;;FA;;;SY)(A;;FA;;;BA)")
	if e != nil {
		t.Fatal(e)
	}
	acl, _, e := descriptor.DACL()
	if e != nil {
		t.Fatal(e)
	}
	if e = windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); e != nil {
		t.Fatal(e)
	}
}
