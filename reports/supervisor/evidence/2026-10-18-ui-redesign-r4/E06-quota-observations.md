# R4 independent quota observations

Tuple order: (Claude5h, ClaudeWeek, Gemini5h, GeminiWeek). -1 means unknown. Fresh on 5996891479f8271bfdbe5822f57dbd81b7161d0f.

| Case | Stage | Actual tuple | HasData | Match |
|---|---|---|---|---|
| full | summary_parser | (0.46, 0.12, 0.88, 0.9) | True | True |
| full | HTTP_pool | (0.46, 0.12, 0.88, 0.9) | True | True |
| full | disk_reload | (0.46, 0.12, 0.88, 0.9) | True | True |
| empty_rest | summary_parser | (-1, -1, -1, -1) | False | True |
| empty_rest | HTTP_pool | (-1, -1, -1, -1) | False | True |
| empty_rest | disk_reload | (-1, -1, -1, -1) | False | True |
| empty_rpc | summary_parser | (-1, -1, -1, -1) | False | True |
| empty_rpc | HTTP_pool | (-1, -1, -1, -1) | False | True |
| empty_rpc | disk_reload | (-1, -1, -1, -1) | False | True |
| missing_fraction | summary_parser | (-1, -1, -1, -1) | False | True |
| missing_fraction | HTTP_pool | (-1, -1, -1, -1) | False | True |
| missing_fraction | disk_reload | (-1, -1, -1, -1) | False | True |
| null_fraction | summary_parser | (-1, -1, -1, -1) | False | True |
| null_fraction | HTTP_pool | (-1, -1, -1, -1) | False | True |
| null_fraction | disk_reload | (-1, -1, -1, -1) | False | True |
| zero_5h_standard_id | summary_parser | (0, -1, -1, -1) | True | True |
| zero_5h_standard_id | HTTP_pool | (0, -1, -1, -1) | True | True |
| zero_5h_standard_id | disk_reload | (0, -1, -1, -1) | True | True |
| zero_week_standard_id | summary_parser | (-1, 0, -1, -1) | True | True |
| zero_week_standard_id | HTTP_pool | (-1, 0, -1, -1) | True | True |
| zero_week_standard_id | disk_reload | (-1, 0, -1, -1) | True | True |
| zero_5h_window_only | summary_parser | (0, -1, -1, -1) | True | True |
| zero_5h_window_only | HTTP_pool | (0, -1, -1, -1) | True | True |
| zero_5h_window_only | disk_reload | (0, -1, -1, -1) | True | True |
| zero_week_window_only | summary_parser | (-1, 0, -1, -1) | True | True |
| zero_week_window_only | HTTP_pool | (-1, 0, -1, -1) | True | True |
| zero_week_window_only | disk_reload | (-1, 0, -1, -1) | True | True |
| nonzero_week_window_only | summary_parser | (-1, -1, -1, 0.37) | True | True |
| nonzero_week_window_only | HTTP_pool | (-1, -1, -1, 0.37) | True | True |
| nonzero_week_window_only | disk_reload | (-1, -1, -1, 0.37) | True | True |
| mixed_missing | summary_parser | (0.23, -1, -1, -1) | True | True |
| mixed_missing | HTTP_pool | (0.23, -1, -1, -1) | True | True |
| mixed_missing | disk_reload | (0.23, -1, -1, -1) | True | True |
| legacy_REST_missing_fraction | HTTP_pool | (-1, -1, -1, -1) | False | True |
| legacy_REST_missing_fraction | disk_reload | (-1, -1, -1, -1) | False | True |
| legacy_REST_zero_no_cycle | HTTP_pool | (0, -1, -1, -1) | True | True |
| legacy_REST_zero_no_cycle | disk_reload | (0, -1, -1, -1) | True | True |
| legacy_REST_amount_camel | HTTP_pool | (-1, -1, -1, -1) | False | True |
| legacy_REST_amount_camel | disk_reload | (-1, -1, -1, -1) | False | True |
| legacy_REST_amount_snake | HTTP_pool | (-1, -1, -1, -1) | False | True |
| legacy_REST_amount_snake | disk_reload | (-1, -1, -1, -1) | False | True |
| legacy_REST_zero_amount | HTTP_pool | (-1, -1, -1, -1) | False | True |
| legacy_REST_zero_amount | disk_reload | (-1, -1, -1, -1) | False | True |
| legacy_REST_null_fraction | HTTP_pool | (-1, -1, -1, -1) | False | True |
| legacy_REST_null_fraction | disk_reload | (-1, -1, -1, -1) | False | True |
| legacy_REST_unknown_then_valid | HTTP_pool | (0.2, -1, -1, -1) | True | True |
| legacy_REST_unknown_then_valid | disk_reload | (0.2, -1, -1, -1) | True | True |
| legacy_REST_valid_then_unknown | HTTP_pool | (0.2, -1, -1, -1) | True | True |
| legacy_REST_valid_then_unknown | disk_reload | (0.2, -1, -1, -1) | True | True |
| both_zero_without_resets | summary_parser | (0, 0, -1, -1) | True | True |
| both_zero_without_resets | HTTP_pool | (0, 0, -1, -1) | True | True |
| both_zero_without_resets | disk_reload | (0, 0, -1, -1) | True | True |
| gemini_zero_week_window_only | summary_parser | (-1, -1, -1, 0) | True | True |
| gemini_zero_week_window_only | HTTP_pool | (-1, -1, -1, 0) | True | True |
| gemini_zero_week_window_only | disk_reload | (-1, -1, -1, 0) | True | True |
| legacy_cache_nonzero | disk_reload | (0.2, -1, 0.8, -1) | True | True |
| legacy_cache_nonzero | disk_resave_reload | (0.2, -1, 0.8, -1) | True | True |
| legacy_cache_zero | disk_reload | (0, -1, 0.8, -1) | True | True |
| legacy_cache_zero | disk_resave_reload | (0, -1, 0.8, -1) | True | True |
| legacy_cache_missing_model | disk_reload | (0.2, -1, -1, -1) | True | True |
| legacy_cache_missing_model | disk_resave_reload | (0.2, -1, -1, -1) | True | True |
| legacy_cache_missing_fraction | disk_reload | (-1, -1, -1, -1) | False | True |
| legacy_cache_missing_fraction | disk_resave_reload | (-1, -1, -1, -1) | False | True |
| legacy_cache_missing_claude | disk_reload | (-1, -1, 0.8, -1) | True | True |
| legacy_cache_missing_claude | disk_resave_reload | (-1, -1, 0.8, -1) | True | True |
| legacy_cache_null_family | disk_reload | (0.2, -1, -1, -1) | True | True |
| legacy_cache_null_family | disk_resave_reload | (0.2, -1, -1, -1) | True | True |
| cache_no_families | disk_reload | (-1, -1, -1, -1) | False | True |
| cache_no_families | disk_resave_reload | (-1, -1, -1, -1) | False | True |
| cache_null_fraction | disk_reload | (-1, -1, -1, -1) | False | True |
| cache_null_fraction | disk_resave_reload | (-1, -1, -1, -1) | False | True |
| cache_weekly_zero_without_raw | disk_reload | (-1, 0, -1, -1) | True | True |
| cache_weekly_zero_without_raw | disk_resave_reload | (-1, 0, -1, -1) | True | True |
| cache_both_zero_without_raw | disk_reload | (0, -1, -1, -1) | True | False |
| cache_both_zero_without_raw | disk_resave_reload | (0, -1, -1, -1) | True | False |
| cache_gemini_both_zero_without_raw | disk_reload | (-1, -1, 0, -1) | True | False |
| cache_gemini_both_zero_without_raw | disk_resave_reload | (-1, -1, 0, -1) | True | False |
| cache_camel_weekly_zero | disk_reload | (-1, 0, -1, -1) | True | True |
| cache_camel_weekly_zero | disk_resave_reload | (-1, 0, -1, -1) | True | True |

79 observations across 33 fixtures: 75 matched, 4 mismatched (two families, initial load and save/reload). The raw JSON includes expected values, input, actual badge text, and stored cycle fields.
