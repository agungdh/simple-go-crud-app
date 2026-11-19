package upload

import "time"

const (
	// EndpointMaxBytes is the fixed per-request limit for POST /upload. It is
	// deliberately not configurable through the environment; the global hard
	// limit still applies when it is lower.
	EndpointMaxBytes int64 = 2 << 20 // 2 MiB
	// TempDir is the pending-upload directory. Files are moved elsewhere once
	// they are attached to a permanent record.
	TempDir = "persistence/upload/tmp"
	// ReadTimeout extends the read deadline for the upload route only, so a
	// slow 2 MiB upload does not get cut off by the server's default timeout.
	ReadTimeout = time.Minute

	// copyBufferSize keeps streaming memory usage small regardless of the
	// upload size.
	copyBufferSize = 64 << 10
)
