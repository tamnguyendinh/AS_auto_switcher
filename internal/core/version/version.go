package version

import "fmt"

var (
	// Version là phiên bản chính thức của ứng dụng (Single Source of Truth).
	// Giá trị fallback khi chạy dev nội bộ hoặc "go test ./...".
	Version = "1.0.3.1"

	// GitCommit là short commit hash của Git (ví dụ: "a77efb9"), được inject qua -ldflags lúc build.
	GitCommit = ""
)

// GetVersion trả về chuỗi phiên bản thuần túy (ví dụ: "1.0.2.0").
func GetVersion() string {
	return Version
}

// GetFullVersion trả về chuỗi phiên bản định dạng đầy đủ hiển thị trên UI.
// Luôn trả về định dạng phiên bản thương mại sạch: "Version: 1.0.2.0", tuyệt đối không nối commit hash.
func GetFullVersion() string {
	return fmt.Sprintf("Version: %s", Version)
}

// GetBuildInfo trả về đầy đủ thông tin kỹ thuật gồm cả Git commit (nếu có) phục vụ logging và chẩn đoán.
func GetBuildInfo() string {
	if GitCommit != "" {
		return fmt.Sprintf("Version %s (commit %s)", Version, GitCommit)
	}
	return fmt.Sprintf("Version %s", Version)
}
