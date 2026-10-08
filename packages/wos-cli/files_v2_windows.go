package woscli

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"unsafe"
)

func validateRegularV2(file *os.File) error {
	info, e := file.Stat()
	if e != nil {
		return e
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("local v2 file must be regular")
	}
	var handle windows.ByHandleFileInformation
	if e = windows.GetFileInformationByHandle(windows.Handle(file.Fd()), &handle); e != nil {
		return e
	}
	if handle.NumberOfLinks != 1 || handle.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return fmt.Errorf("hardlinks/reparse files are forbidden")
	}
	return nil
}
func validateSecretFile(file *os.File) error {
	if e := validateRegularV2(file); e != nil {
		return e
	}
	sd, e := windows.GetSecurityInfo(windows.Handle(file.Fd()), windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if e != nil {
		return fmt.Errorf("mounted secret ACL unavailable")
	}
	acl, _, e := sd.DACL()
	if e != nil || acl == nil {
		return fmt.Errorf("mounted secret must have a restricted DACL")
	}
	user, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		return fmt.Errorf("secret owner identity unavailable")
	}
	system, e := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if e != nil {
		return e
	}
	admins, e := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if e != nil {
		return e
	}
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if e = windows.GetAce(acl, i, &ace); e != nil {
			return fmt.Errorf("mounted secret ACL unavailable")
		}
		if ace.Header.AceType == windows.ACCESS_DENIED_ACE_TYPE {
			continue
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			return fmt.Errorf("mounted secret uses an unsupported ACL entry")
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if !sid.Equals(user.User.Sid) && !sid.Equals(system) && !sid.Equals(admins) {
			return fmt.Errorf("mounted secret grants another identity access")
		}
	}
	return nil
}
