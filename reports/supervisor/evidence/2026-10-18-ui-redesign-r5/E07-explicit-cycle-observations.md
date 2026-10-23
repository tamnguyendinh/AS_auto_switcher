# R5 explicit-cycle observations

Source: E06-runtime-probe.json. Synthetic normalized JSON/public parser -> two fresh disk-cache reloads -> production extractor and NewAccountCardWithQuota.

Tuple order: Claude5h, ClaudeWeek, Gemini5h, GeminiWeek. -1 means unknown.

| Case | Stage | Stored cycles | Legacy C/G | Extracted cycles | HasData | 5h row | Week row | Match |
|---|---|---|---|---|---|---|---|---|
| claude-parser-zero | normalized_input | (0,0,-1,-1) | (0,-1) | (0,0,-1,-1) | True | 5h: Claude 0% Gemini --% | week: Claude 0% Gemini --% | True |
| claude-parser-zero | save_reload_1 | (0,0,-1,-1) | (0,-1) | (0,0,-1,-1) | True | 5h: Claude 0% Gemini --% | week: Claude 0% Gemini --% | True |
| claude-parser-zero | save_reload_2 | (0,0,-1,-1) | (0,-1) | (0,0,-1,-1) | True | 5h: Claude 0% Gemini --% | week: Claude 0% Gemini --% | True |
| gemini-parser-zero | normalized_input | (-1,-1,0,0) | (-1,0) | (-1,-1,0,0) | True | 5h: Claude --% Gemini 0% | week: Claude --% Gemini 0% | True |
| gemini-parser-zero | save_reload_1 | (-1,-1,0,0) | (-1,0) | (-1,-1,0,0) | True | 5h: Claude --% Gemini 0% | week: Claude --% Gemini 0% | True |
| gemini-parser-zero | save_reload_2 | (-1,-1,0,0) | (-1,0) | (-1,-1,0,0) | True | 5h: Claude --% Gemini 0% | week: Claude --% Gemini 0% | True |
| both-parser-zero | normalized_input | (0,0,0,0) | (0,0) | (0,0,0,0) | True | 5h: Claude 0% Gemini 0% | week: Claude 0% Gemini 0% | True |
| both-parser-zero | save_reload_1 | (0,0,0,0) | (0,0) | (0,0,0,0) | True | 5h: Claude 0% Gemini 0% | week: Claude 0% Gemini 0% | True |
| both-parser-zero | save_reload_2 | (0,0,0,0) | (0,0) | (0,0,0,0) | True | 5h: Claude 0% Gemini 0% | week: Claude 0% Gemini 0% | True |
| both-explicit-zero | normalized_input | (0,0,0,0) | (0,0) | (0,0,0,0) | True | 5h: Claude 0% Gemini 0% | week: Claude 0% Gemini 0% | True |
| both-explicit-zero | save_reload_1 | (0,0,0,0) | (0,0) | (0,0,0,0) | True | 5h: Claude 0% Gemini 0% | week: Claude 0% Gemini 0% | True |
| both-explicit-zero | save_reload_2 | (0,0,0,0) | (0,0) | (0,0,0,0) | True | 5h: Claude 0% Gemini 0% | week: Claude 0% Gemini 0% | True |
| claude-no-legacy-fraction | normalized_input | (0,0,-1,-1) | (-1,-1) | (0,0,-1,-1) | True | 5h: Claude 0% Gemini --% | week: Claude 0% Gemini --% | True |
| claude-no-legacy-fraction | save_reload_1 | (0,0,-1,-1) | (-1,-1) | (0,0,-1,-1) | True | 5h: Claude 0% Gemini --% | week: Claude 0% Gemini --% | True |
| claude-no-legacy-fraction | save_reload_2 | (0,0,-1,-1) | (-1,-1) | (0,0,-1,-1) | True | 5h: Claude 0% Gemini --% | week: Claude 0% Gemini --% | True |
| gemini-no-legacy-fraction | normalized_input | (-1,-1,0,0) | (-1,-1) | (-1,-1,0,0) | True | 5h: Claude --% Gemini 0% | week: Claude --% Gemini 0% | True |
| gemini-no-legacy-fraction | save_reload_1 | (-1,-1,0,0) | (-1,-1) | (-1,-1,0,0) | True | 5h: Claude --% Gemini 0% | week: Claude --% Gemini 0% | True |
| gemini-no-legacy-fraction | save_reload_2 | (-1,-1,0,0) | (-1,-1) | (-1,-1,0,0) | True | 5h: Claude --% Gemini 0% | week: Claude --% Gemini 0% | True |
| both-camel-zero | normalized_input | (0,0,0,0) | (0,0) | (0,0,0,0) | True | 5h: Claude 0% Gemini 0% | week: Claude 0% Gemini 0% | True |
| both-camel-zero | save_reload_1 | (0,0,0,0) | (0,0) | (0,0,0,0) | True | 5h: Claude 0% Gemini 0% | week: Claude 0% Gemini 0% | True |
| both-camel-zero | save_reload_2 | (0,0,0,0) | (0,0) | (0,0,0,0) | True | 5h: Claude 0% Gemini 0% | week: Claude 0% Gemini 0% | True |
| explicit-unknown | normalized_input | (-1,-1,-1,-1) | (-1,-1) | (-1,-1,-1,-1) | False | 5h: Claude --% Gemini --% | week: Claude --% Gemini --% | True |
| explicit-unknown | save_reload_1 | (-1,-1,-1,-1) | (-1,-1) | (-1,-1,-1,-1) | False | 5h: Claude --% Gemini --% | week: Claude --% Gemini --% | True |
| explicit-unknown | save_reload_2 | (-1,-1,-1,-1) | (-1,-1) | (-1,-1,-1,-1) | False | 5h: Claude --% Gemini --% | week: Claude --% Gemini --% | True |
| missing-families | normalized_input | (-1,-1,-1,-1) | (-1,-1) | (-1,-1,-1,-1) | False | 5h: Claude --% Gemini --% | week: Claude --% Gemini --% | True |
| missing-families | save_reload_1 | (-1,-1,-1,-1) | (-1,-1) | (-1,-1,-1,-1) | False | 5h: Claude --% Gemini --% | week: Claude --% Gemini --% | True |
| missing-families | save_reload_2 | (-1,-1,-1,-1) | (-1,-1) | (-1,-1,-1,-1) | False | 5h: Claude --% Gemini --% | week: Claude --% Gemini --% | True |
| null-families | normalized_input | (-1,-1,-1,-1) | (-1,-1) | (-1,-1,-1,-1) | False | 5h: Claude --% Gemini --% | week: Claude --% Gemini --% | True |
| null-families | save_reload_1 | (-1,-1,-1,-1) | (-1,-1) | (-1,-1,-1,-1) | False | 5h: Claude --% Gemini --% | week: Claude --% Gemini --% | True |
| null-families | save_reload_2 | (-1,-1,-1,-1) | (-1,-1) | (-1,-1,-1,-1) | False | 5h: Claude --% Gemini --% | week: Claude --% Gemini --% | True |
| legacy-only-zero | normalized_input | (-1,-1,-1,-1) | (0,-1) | (0,-1,-1,-1) | True | 5h: Claude 0% Gemini --% | week: Claude --% Gemini --% | True |
| legacy-only-zero | save_reload_1 | (-1,-1,-1,-1) | (0,-1) | (0,-1,-1,-1) | True | 5h: Claude 0% Gemini --% | week: Claude --% Gemini --% | True |
| legacy-only-zero | save_reload_2 | (-1,-1,-1,-1) | (0,-1) | (0,-1,-1,-1) | True | 5h: Claude 0% Gemini --% | week: Claude --% Gemini --% | True |
| legacy-only-positive | normalized_input | (-1,-1,-1,-1) | (0.2,-1) | (0.2,-1,-1,-1) | True | 5h: Claude 20% Gemini --% | week: Claude --% Gemini --% | True |
| legacy-only-positive | save_reload_1 | (-1,-1,-1,-1) | (0.2,-1) | (0.2,-1,-1,-1) | True | 5h: Claude 20% Gemini --% | week: Claude --% Gemini --% | True |
| legacy-only-positive | save_reload_2 | (-1,-1,-1,-1) | (0.2,-1) | (0.2,-1,-1,-1) | True | 5h: Claude 20% Gemini --% | week: Claude --% Gemini --% | True |
| weekly-only-zero | normalized_input | (-1,0,-1,-1) | (-1,-1) | (-1,0,-1,-1) | True | 5h: Claude --% Gemini --% | week: Claude 0% Gemini --% | True |
| weekly-only-zero | save_reload_1 | (-1,0,-1,-1) | (-1,-1) | (-1,0,-1,-1) | True | 5h: Claude --% Gemini --% | week: Claude 0% Gemini --% | True |
| weekly-only-zero | save_reload_2 | (-1,0,-1,-1) | (-1,-1) | (-1,0,-1,-1) | True | 5h: Claude --% Gemini --% | week: Claude 0% Gemini --% | True |
| five-hour-only-zero | normalized_input | (-1,-1,0,-1) | (-1,-1) | (-1,-1,0,-1) | True | 5h: Claude --% Gemini 0% | week: Claude --% Gemini --% | True |
| five-hour-only-zero | save_reload_1 | (-1,-1,0,-1) | (-1,-1) | (-1,-1,0,-1) | True | 5h: Claude --% Gemini 0% | week: Claude --% Gemini --% | True |
| five-hour-only-zero | save_reload_2 | (-1,-1,0,-1) | (-1,-1) | (-1,-1,0,-1) | True | 5h: Claude --% Gemini 0% | week: Claude --% Gemini --% | True |
| claude-legacy-positive | normalized_input | (0,0,-1,-1) | (0.2,-1) | (0.2,-1,-1,-1) | True | 5h: Claude 20% Gemini --% | week: Claude --% Gemini --% | False |
| claude-legacy-positive | save_reload_1 | (0,0,-1,-1) | (0.2,-1) | (0.2,-1,-1,-1) | True | 5h: Claude 20% Gemini --% | week: Claude --% Gemini --% | False |
| claude-legacy-positive | save_reload_2 | (0,0,-1,-1) | (0.2,-1) | (0.2,-1,-1,-1) | True | 5h: Claude 20% Gemini --% | week: Claude --% Gemini --% | False |
| gemini-legacy-positive | normalized_input | (-1,-1,0,0) | (-1,0.8) | (-1,-1,0.8,-1) | True | 5h: Claude --% Gemini 80% | week: Claude --% Gemini --% | False |
| gemini-legacy-positive | save_reload_1 | (-1,-1,0,0) | (-1,0.8) | (-1,-1,0.8,-1) | True | 5h: Claude --% Gemini 80% | week: Claude --% Gemini --% | False |
| gemini-legacy-positive | save_reload_2 | (-1,-1,0,0) | (-1,0.8) | (-1,-1,0.8,-1) | True | 5h: Claude --% Gemini 80% | week: Claude --% Gemini --% | False |
| both-legacy-positive | normalized_input | (0,0,0,0) | (0.2,0.8) | (0.2,-1,0.8,-1) | True | 5h: Claude 20% Gemini 80% | week: Claude --% Gemini --% | False |
| both-legacy-positive | save_reload_1 | (0,0,0,0) | (0.2,0.8) | (0.2,-1,0.8,-1) | True | 5h: Claude 20% Gemini 80% | week: Claude --% Gemini --% | False |
| both-legacy-positive | save_reload_2 | (0,0,0,0) | (0.2,0.8) | (0.2,-1,0.8,-1) | True | 5h: Claude 20% Gemini 80% | week: Claude --% Gemini --% | False |
| claude-positive-with-reset | normalized_input | (0,0,-1,-1) | (0.2,-1) | (0,0,-1,-1) | True | 5h: Claude 0% Gemini --% | week: Claude 0% Gemini --% | True |
| claude-positive-with-reset | save_reload_1 | (0,0,-1,-1) | (0.2,-1) | (0,0,-1,-1) | True | 5h: Claude 0% Gemini --% | week: Claude 0% Gemini --% | True |
| claude-positive-with-reset | save_reload_2 | (0,0,-1,-1) | (0.2,-1) | (0,0,-1,-1) | True | 5h: Claude 0% Gemini --% | week: Claude 0% Gemini --% | True |
| gemini-positive-with-reset | normalized_input | (-1,-1,0,0) | (-1,0.8) | (-1,-1,0,0) | True | 5h: Claude --% Gemini 0% | week: Claude --% Gemini 0% | True |
| gemini-positive-with-reset | save_reload_1 | (-1,-1,0,0) | (-1,0.8) | (-1,-1,0,0) | True | 5h: Claude --% Gemini 0% | week: Claude --% Gemini 0% | True |
| gemini-positive-with-reset | save_reload_2 | (-1,-1,0,0) | (-1,0.8) | (-1,-1,0,0) | True | 5h: Claude --% Gemini 0% | week: Claude --% Gemini 0% | True |
| claude-positive-with-raw | normalized_input | (0,0,-1,-1) | (0.2,-1) | (0,0,-1,-1) | True | 5h: Claude 0% Gemini --% | week: Claude 0% Gemini --% | True |
| claude-positive-with-raw | save_reload_1 | (0,0,-1,-1) | (0.2,-1) | (0,0,-1,-1) | True | 5h: Claude 0% Gemini --% | week: Claude 0% Gemini --% | True |
| claude-positive-with-raw | save_reload_2 | (0,0,-1,-1) | (0.2,-1) | (0,0,-1,-1) | True | 5h: Claude 0% Gemini --% | week: Claude 0% Gemini --% | True |

E07-explicit-cycle-cards.png is a Fyne software-canvas capture (not a native OS screenshot).
Cards in row-major order: Claude parser zero, Gemini parser zero, both explicit zero; explicit unknown, legacy-only zero, weekly-only zero; Claude legacy positive, Gemini legacy positive, both legacy positive.
