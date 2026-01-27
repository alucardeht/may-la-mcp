package version

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

const ProtocolVersion = "2025-11-25"

var SupportedProtocolVersions = []string{"2025-11-25", "2024-11-05"}
