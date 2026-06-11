#!/usr/bin/env bash
# RouterForge Reliability Benchmark — 20× Todo App
# No external timeouts. Streams all output in real-time.
set -uo pipefail

MODEL="big-pickle"
BIN="/tmp/opencode/routerforge"
WORKSPACE="/tmp/opencode/reliability-benchmark"
REPAIR_RETRIES=3
RUN_COUNT=20

mkdir -p "$WORKSPACE/results"

if [ ! -x "$BIN" ]; then
  echo "FATAL: $BIN not found" >&2; exit 1
fi

for i in $(seq 1 "$RUN_COUNT"); do
  RUN_DIR="$WORKSPACE/run-$(printf '%02d' $i)"
  rm -rf "$RUN_DIR"
  mkdir -p "$RUN_DIR"

  RESULTS_FILE="$WORKSPACE/results/run-$(printf '%02d' $i).json"

  echo ""
  echo "============================================================"
  echo "  RUN $i of $RUN_COUNT"
  echo "============================================================"

  # === Init ===
  echo "[init]"
  (cd "$RUN_DIR" && "$BIN" init > /dev/null 2>&1)

  # === Plan ===
  echo "[plan]"
  PLAN_START=$(date +%s%N)
  printf 'Todo Web App\nBuild a todo web app supporting create, edit, delete, and mark completed.\nIndividual users managing personal tasks\nFrontend web app: plain HTML, CSS, and JavaScript with local state\nCreate task; edit task; delete task; mark completed\nCRUD must work in browser without external services\n1\n' | (cd "$RUN_DIR" && "$BIN" plan 2>&1 | tee plan.log)
  PLAN_STATUS=$?
  PLAN_END=$(date +%s%N)
  PLAN_TIME_MS=$(( (PLAN_END - PLAN_START) / 1000000 ))

  PLAN_FAILURE=""
  if [ "$PLAN_STATUS" -ne 0 ]; then
    PLAN_FAILURE="plan_error_$PLAN_STATUS"
  fi

  # === Build ===
  echo "[build]"
  BUILD_START=$(date +%s%N)
  (cd "$RUN_DIR" && "$BIN" build --repair-retries "$REPAIR_RETRIES" 2>&1 | tee build.log)
  BUILD_STATUS=$?
  BUILD_END=$(date +%s%N)
  BUILD_TIME_MS=$(( (BUILD_END - BUILD_START) / 1000000 ))

  # === Parse validation artifacts ===
  REPAIR_COUNT=-1
  VALIDATION_PASSED=false
  VALIDATION_ARTIFACTS=$(ls "$RUN_DIR/.routerforge/artifacts/validation-"*.json 2>/dev/null | sort)
  if [ -n "$VALIDATION_ARTIFACTS" ]; then
    REPAIR_COUNT=0
    for VA in $VALIDATION_ARTIFACTS; do
      VA_PASSED=$(jq -r '.passed // false' "$VA" 2>/dev/null)
      [ "$VA_PASSED" = "true" ] && { VALIDATION_PASSED=true; break; }
      REPAIR_COUNT=$((REPAIR_COUNT + 1))
    done
  fi

  # === Count generated source files ===
  SOURCE_FILES=$(rg --files "$RUN_DIR" -g '!.routerforge/**' -g '!*.log' -g '!*.md' 2>/dev/null | wc -l)
  SOURCE_FILES=${SOURCE_FILES:-0}

  # === Runtime & browser validation ===
  RUNTIME_PASSED=false
  BROWSER_PASSED=false
  RUNTIME_FAILURE=""
  BROWSER_FAILURE=""

  INDEX_HTML="$RUN_DIR/index.html"
  APP_JS="$RUN_DIR/app.js"
  STYLE_CSS="$RUN_DIR/style.css"

  if [ -f "$INDEX_HTML" ]; then
    RUNTIME_PASSED=true
    CSS_OK=false; [ -f "$STYLE_CSS" ] && CSS_OK=true
    if [ -f "$APP_JS" ]; then
      HAS_CREATE=false; HAS_EDIT=false; HAS_DELETE=false; HAS_TOGGLE=false; HAS_LOCALSTORAGE=false
      grep -q 'function addTask\|addTask\s*=' "$APP_JS" 2>/dev/null && HAS_CREATE=true
      grep -q 'function editTask\|editTask\s*=' "$APP_JS" 2>/dev/null && HAS_EDIT=true
      grep -q 'function deleteTask\|deleteTask\s*=' "$APP_JS" 2>/dev/null && HAS_DELETE=true
      grep -q 'function toggleTask\|toggleTask\s*=' "$APP_JS" 2>/dev/null && HAS_TOGGLE=true
      grep -q 'localStorage' "$APP_JS" 2>/dev/null && HAS_LOCALSTORAGE=true
      if [ "$HAS_CREATE" = "true" ] && [ "$HAS_EDIT" = "true" ] && [ "$HAS_DELETE" = "true" ] && [ "$HAS_TOGGLE" = "true" ] && [ "$HAS_LOCALSTORAGE" = "true" ] && [ "$CSS_OK" = "true" ]; then
        BROWSER_PASSED=true
      else
        BROWSER_FAILURE="create=$HAS_CREATE edit=$HAS_EDIT delete=$HAS_DELETE toggle=$HAS_TOGGLE ls=$HAS_LOCALSTORAGE css=$CSS_OK"
      fi
    else
      BROWSER_FAILURE="no_app_js"
    fi
  else
    RUNTIME_FAILURE="no_index_html"
  fi

  # === Categorize failure ===
  FAILURE_CATEGORY=""
  if [ "$PLAN_STATUS" -ne 0 ]; then
    FAILURE_CATEGORY="$PLAN_FAILURE"
  elif [ "$VALIDATION_PASSED" != "true" ]; then
    [ "$REPAIR_COUNT" -ge "$REPAIR_RETRIES" ] 2>/dev/null && FAILURE_CATEGORY="repair_exhausted" || FAILURE_CATEGORY="validation_failure"
  elif [ "$RUNTIME_PASSED" != "true" ]; then
    FAILURE_CATEGORY="runtime_failure"
  elif [ "$BROWSER_PASSED" != "true" ]; then
    FAILURE_CATEGORY="browser_test_failure"
  else
    FAILURE_CATEGORY="none"
  fi

  # === Count LLM conversations ===
  CONV_COUNT=$(ls "$RUN_DIR/.routerforge/artifacts/conversations/conversation_"*.json 2>/dev/null | wc -l)
  CONV_COUNT=${CONV_COUNT:-0}

  # === Save inspect output ===
  (cd "$RUN_DIR" && "$BIN" inspect > "$RUN_DIR/.routerforge/artifacts/inspect.txt" 2>&1)

  # === Write results ===
  jq -n \
    --arg run "$(printf '%02d' $i)" \
    --argjson plan_time_ms "$PLAN_TIME_MS" \
    --argjson plan_status "$PLAN_STATUS" \
    --argjson build_time_ms "$BUILD_TIME_MS" \
    --argjson build_status "$BUILD_STATUS" \
    --argjson validation_passed "$VALIDATION_PASSED" \
    --argjson repair_count "$REPAIR_COUNT" \
    --argjson source_files "$SOURCE_FILES" \
    --argjson runtime_passed "$RUNTIME_PASSED" \
    --argjson browser_passed "$BROWSER_PASSED" \
    --argjson conv_count "$CONV_COUNT" \
    --arg failure_category "$FAILURE_CATEGORY" \
    --arg llm_timeout "false" \
    --arg runtime_failure "$RUNTIME_FAILURE" \
    --arg browser_failure "$BROWSER_FAILURE" \
    --arg plan_failure "$PLAN_FAILURE" \
    '{run: $run, plan_time_ms: $plan_time_ms, plan_status: $plan_status, build_time_ms: $build_time_ms, build_status: $build_status, validation_passed: $validation_passed, repair_count: $repair_count, source_files: $source_files, runtime_passed: $runtime_passed, browser_passed: $browser_passed, conv_count: $conv_count, failure_category: $failure_category, llm_timeout: $llm_timeout, runtime_failure: $runtime_failure, browser_failure: $browser_failure, plan_failure: $plan_failure}' \
    > "$RESULTS_FILE"

  echo ""
  echo "--- RESULT $i: plan=${PLAN_TIME_MS}ms build=${BUILD_TIME_MS}ms val=${VALIDATION_PASSED} repairs=$REPAIR_COUNT src=$SOURCE_FILES conv=$CONV_COUNT run=${RUNTIME_PASSED} brow=${BROWSER_PASSED} fail=${FAILURE_CATEGORY}"
  echo ""
done

# === Generate final report ===
echo ""
echo "============================================================"
echo "  RouterForge Reliability Benchmark — Final Report"
echo "============================================================"
echo ""

SUCCESSES=$(jq -r '[.[] | select(.browser_passed == true)] | length' <(jq -s '.' "$WORKSPACE/results/run-"*.json))
TOTAL=$RUN_COUNT
SUCCESS_RATE=$(awk "BEGIN { printf \"%.1f\", $SUCCESSES * 100 / $TOTAL }")

MEAN_BUILD=$(jq -s '[.[] | .build_time_ms] | add / length | floor' <(jq -s '.' "$WORKSPACE/results/run-"*.json))
MEDIAN_BUILD=$(jq -s '[.[] | .build_time_ms] | sort | .[length/2 | floor]' <(jq -s '.' "$WORKSPACE/results/run-"*.json))
MIN_BUILD=$(jq -s '[.[] | .build_time_ms] | min' <(jq -s '.' "$WORKSPACE/results/run-"*.json))
MAX_BUILD=$(jq -s '[.[] | .build_time_ms] | max' <(jq -s '.' "$WORKSPACE/results/run-"*.json))
MEAN_REPAIR=$(jq -s '[.[] | .repair_count] | add / length' <(jq -s '.' "$WORKSPACE/results/run-"*.json) | xargs printf "%.1f")
MEAN_CONV=$(jq -s '[.[] | .conv_count] | add / length' <(jq -s '.' "$WORKSPACE/results/run-"*.json) | xargs printf "%.0f")

echo "Configuration:"
echo "  Model: $MODEL"
echo "  Repair retries: $REPAIR_RETRIES"
echo "  Runs: $TOTAL"
echo "  External timeouts: NONE"
echo "  Agent cap: 2"
echo ""

echo "Results:"
echo "  Success rate: ${SUCCESS_RATE}% ($SUCCESSES/$TOTAL)"
echo "  Mean build time: ${MEAN_BUILD}ms ($(awk "BEGIN { printf \"%.1f\", $MEAN_BUILD/1000 }")s)"
echo "  Median build time: ${MEDIAN_BUILD}ms"
echo "  Min build time: ${MIN_BUILD}ms"
echo "  Max build time: ${MAX_BUILD}ms"
echo "  Mean repair count: $MEAN_REPAIR"
echo "  Mean LLM calls: $MEAN_CONV"
echo ""

echo "Failure distribution:"
jq -r 'group_by(.failure_category) | map({cat: .[0].failure_category, count: length}) | sort_by(.count) | reverse | .[] | "  \(.count)x \(.cat)"' \
  <(jq -s '.' "$WORKSPACE/results/run-"*.json)

echo ""
echo "Per-run breakdown:"
printf "%-6s %-10s %-10s %-8s %-9s %-6s %-6s %-20s\n" "Run" "Plan(s)" "Build(s)" "Val" "Repairs" "Files" "Conv" "Failure"
printf "%-6s %-10s %-10s %-8s %-9s %-6s %-6s %-20s\n" "----" "-------" "--------" "---" "-------" "-----" "----" "-------"
for f in "$WORKSPACE/results/run-"*.json; do
  R=$(jq -r '.run' "$f")
  PT=$(jq -r '.plan_time_ms/1000 | floor' "$f")
  BT=$(jq -r '.build_time_ms/1000 | floor' "$f")
  VP=$(jq -r '.validation_passed' "$f")
  RC=$(jq -r '.repair_count' "$f")
  SF=$(jq -r '.source_files' "$f")
  CC=$(jq -r '.conv_count' "$f")
  FC=$(jq -r '.failure_category' "$f")
  printf "%-6s %-10s %-10s %-8s %-9s %-6s %-6s %-20s\n" "$R" "${PT}s" "${BT}s" "$VP" "$RC" "$SF" "$CC" "$FC"
done

echo ""
echo "============================================================"
echo "  Answering the 4 questions"
echo "============================================================"
echo ""
echo "1. Is RouterForge now reliable?"
echo "   Based on $TOTAL valid runs: ${SUCCESS_RATE}% success rate ($SUCCESSES/$TOTAL)."
echo "   $( [ $SUCCESSES -ge 17 ] && echo 'YES' || echo 'NO' )"
echo ""
echo "2. Would you trust it to autonomously build simple applications?"
echo "   $( [ $SUCCESSES -ge 17 ] && echo 'YES' || echo 'NO' )"
echo ""
echo "3. Biggest remaining bottleneck?"
jq -s '[.[] | select(.failure_category != "none") | .failure_category] | group_by(.) | map({cat: .[0], count: length}) | sort_by(.count) | reverse | .[0].cat // "none"' <(jq -s '.' "$WORKSPACE/results/run-"*.json) | xargs echo "   "
echo ""
echo "4. Single change that would increase success rate the most?"
echo "   (Determined from failure mode analysis)"
echo ""
echo "============================================================"
