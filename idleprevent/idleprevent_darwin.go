//go:build darwin

package idleprevent

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include <stdlib.h>
#include <IOKit/pwr_mgt/IOPMLib.h>
#include <CoreFoundation/CoreFoundation.h>

static IOPMAssertionID yamusicIdleAssertion;

int yamusicIdleAcquire(const char *reason) {
	CFStringRef r = CFStringCreateWithCString(kCFAllocatorDefault, reason, kCFStringEncodingUTF8);
	if (!r) {
		return -1;
	}
	IOReturn rc = IOPMAssertionCreateWithName(
		kIOPMAssertionTypePreventUserIdleSystemSleep,
		kIOPMAssertionLevelOn,
		r,
		&yamusicIdleAssertion);
	CFRelease(r);
	if (rc != kIOReturnSuccess) {
		return -1;
	}
	return 0;
}

void yamusicIdleRelease(void) {
	if (yamusicIdleAssertion == 0) {
		return;
	}
	IOPMAssertionRelease(yamusicIdleAssertion);
	yamusicIdleAssertion = 0;
}
*/
import "C"
import (
	"unsafe"

	"github.com/dece2183/yamusic-tui/log"
)

func Supported() bool { return true }

func acquire() bool {
	cs := C.CString("yamusic-tui always-on")
	defer C.free(unsafe.Pointer(cs))
	if C.yamusicIdleAcquire(cs) != 0 {
		log.Print(log.LVL_WARNIGN, "idleprevent IOPMAssertion failed")
		return false
	}
	return true
}

func release() {
	C.yamusicIdleRelease()
}
