package upload

import (
	"testing"
	"time"
)

func TestFixedEndpointSettings(t *testing.T) {
	if EndpointMaxBytes != 2<<20 {
		t.Errorf("EndpointMaxBytes = %d, want %d", EndpointMaxBytes, int64(2<<20))
	}
	if TempDir != "persistence/upload/tmp" {
		t.Errorf("TempDir = %q, want persistence/upload/tmp", TempDir)
	}
	if ReadTimeout != time.Minute {
		t.Errorf("ReadTimeout = %s, want 1m", ReadTimeout)
	}
}
