package version

import "fmt"

var (
	// Version là phiên bản chính thức của ứng dụng (Single Source of Truth).
	// Giá trị fallback khi chạy dev nội bộ hoặc "go test ./...".
	Version = "1.0.2.0"

	// GitCommit là short commit hash của Git (ví dụ: "a77efb9"), được inject qua -ldflags lúc build.
	GitCommit = ""
)

// GetVersion trả về chuỗi phiên bản thuần túy (ví dụ: "1.0.2.0").
func GetVersion() string {
	return Version
}

// GetFullVersion trả về chuỗi phiên bản định dạng đầy đủ hiển thị trên UI.
// Nếu có GitCommit: "Version: 1.0.2.0 (rev a77efb9)"
// Nếu không có GitCommit: "Version: 1.0.2.0"
func GetFullVersion() string {
	if GitCommit != "" {
		return fmt.Sprintf("Version: %s (rev %s)", Version, GitCommit)
	}
	return fmt.Sprintf("Version: %s", Version)
}
