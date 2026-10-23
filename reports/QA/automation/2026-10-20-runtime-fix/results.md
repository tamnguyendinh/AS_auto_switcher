# Runtime repair verification — 2026-10-20

Source baseline: `3d58dff`; working tree chưa commit. Chi tiết: `reports/coder/rp_coder_261020_201600_by_gpt-6_autonomous_switch_runtime_fix.md`.

| Check | Result | Artifact |
|---|---|---|
| CLI và GUI build | PASS; tool exit 0 | Binary hashes trong sha256.json |
| Full package suite | FAIL một test callback GUI; các package khác PASS | full-tests-initial.log |
| UI suite sau sửa condition wait | PASS | ui-tests-after-wait-fix.log |
| Race automation + connector | PASS | automation-connector-race.log |
| Native stream read | Đọc được IDLE/ERROR/429 của tài khoản Owner chỉ định | results.json, provenance là tool output trong cuộc trao đổi |
| Full native GUI switch | Agent chưa capture được; công cụ chụp cửa sổ timeout | Không có screenshot/trace được tuyên bố PASS |
| Phản hồi Owner | App hoạt động tốt theo phản hồi trong chat | results.json |

Không chạy thêm test, switch, prompt hoặc live probe trong lượt viết báo cáo. Log giữ nguyên từ lượt kiểm tra trước.
