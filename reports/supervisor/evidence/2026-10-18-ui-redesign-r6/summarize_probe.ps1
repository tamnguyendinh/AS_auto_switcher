$ErrorActionPreference = 'Stop'
$probe = Get-Content -Raw -LiteralPath (Join-Path $PSScriptRoot 'E06-runtime-probe.json') | ConvertFrom-Json
$rows = foreach ($case in $probe.F1_R6_explicit_zero_matrix) {
    foreach ($observation in $case.observations) {
        [pscustomobject]@{
            case=$case.case
            stage=$observation.stage
            stored_cycles=$observation.stored_cycles
            legacy=$observation.stored_legacy_fractions
            extracted=@($observation.extracted.Claude5h,$observation.extracted.ClaudeWeek,$observation.extracted.Gemini5h,$observation.extracted.GeminiWeek)
            has_data=$observation.extracted.HasData
            raw_count=$observation.raw_count
            resets_absent=$observation.all_resets_absent
            matched=($observation.extract_matches -and $observation.card_matches)
            row_5h=($observation.card_rows.'5h:' -join ' ')
            row_week=($observation.card_rows.'week:' -join ' ')
        }
    }
}
$rows | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $PSScriptRoot 'E07-explicit-cycle-observations.json')
$table = @(
    '# R6 explicit-cycle observations'
    ''
    'Source: E06-runtime-probe.json. Synthetic normalized JSON/public parser -> two fresh disk-cache reloads -> production extractor and NewAccountCardWithQuota.'
    ''
    'Tuple order: Claude5h, ClaudeWeek, Gemini5h, GeminiWeek. -1 means unknown.'
    ''
    '| Case | Stage | Stored cycles | Legacy C/G | Extracted cycles | HasData | 5h row | Week row | Match |'
    '|---|---|---|---|---|---|---|---|---|'
)
foreach ($row in $rows) {
    $table += '| '+$row.case+' | '+$row.stage+' | ('+($row.stored_cycles -join ',')+') | ('+($row.legacy -join ',')+') | ('+($row.extracted -join ',')+') | '+$row.has_data+' | '+$row.row_5h+' | '+$row.row_week+' | '+$row.matched+' |'
}
$table += @(
    ''
    'E07-explicit-cycle-cards.png is a Fyne software-canvas capture (not a native OS screenshot).'
    'Cards in row-major order: Claude parser zero, Gemini parser zero, both explicit zero; explicit unknown, legacy-only zero, weekly-only zero; Claude legacy positive, Gemini legacy positive, both legacy positive.'
)
$table | Set-Content -LiteralPath (Join-Path $PSScriptRoot 'E07-explicit-cycle-observations.md')
$summary = [pscustomobject]@{
    reviewed_commit=$probe.reviewed_commit
    assertions_total=$probe.assertions_total
    assertions_passed=$probe.assertions_passed
    assertion_failures=$probe.assertion_failures
    r5_prior_failures_replayed=@($probe.assertion_results | Where-Object { $_.name -match '^F1/explicit/(claude|gemini|both)-legacy-positive/stage[012]/(extract|card_rows)$' }).Count
    r5_prior_failures_now_pass=@($probe.assertion_results | Where-Object { $_.name -match '^F1/explicit/(claude|gemini|both)-legacy-positive/stage[012]/(extract|card_rows)$' -and $_.pass }).Count
    explicit_fixture_count=@($probe.F1_R6_explicit_zero_matrix).Count
    explicit_observations=@($rows).Count
    explicit_observations_matched=@($rows | Where-Object matched).Count
    explicit_observations_failed=@($rows | Where-Object { -not $_.matched }).Count
    inherited_matrix_case_count=@($probe.F1_R6_HTTP_cache_matrix).Count
    inherited_matrix_observations=@($probe.F1_R6_HTTP_cache_matrix | ForEach-Object { $_.PSObject.Properties | Where-Object { $_.Value.PSObject.Properties.Name -contains 'matches_expected' } }).Count
}
$summary | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $PSScriptRoot 'E06-runtime-summary.json')
$summary | ConvertTo-Json -Depth 5
$rows | Where-Object { $_.stage -eq 'save_reload_2' -and $_.case -match 'legacy-positive|explicit-unknown|raw-' } | ConvertTo-Json -Depth 4 -Compress
$probe.F2_contract | ConvertTo-Json -Compress
$probe.F3_actual_button_refresh | ConvertTo-Json -Depth 4 -Compress
$probe.F4_R6_reflection_AST | ConvertTo-Json -Depth 4 -Compress
$probe.F5_natural_layout_sweep | Select-Object account_count,canvas_size,grid_size,max_card_height,compact_height_matches_min,bottom_scroll_offset | ConvertTo-Json -Depth 4 -Compress
$probe.F6_theme_selector | ConvertTo-Json -Compress

