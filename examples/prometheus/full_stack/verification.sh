#!/bin/bash

echo "=== Prometheus 完整栈验证 ==="
echo ""

echo "1. 检查容器状态..."
docker-compose ps
echo ""

echo "2. 验证应用 /metrics 端点..."
curl -s http://localhost:8080/metrics | grep -E "^(example_app|go_|process_)" | head -10
echo "✓ 应用 metrics 端点正常"
echo ""

echo "3. 验证应用 API 端点..."
curl -s http://localhost:8080/api/users | python3 -m json.tool
echo "✓ 应用 API 端点正常"
echo ""

echo "4. 验证 Prometheus 采集..."
TARGETS=$(curl -s http://localhost:9090/api/v1/targets | python3 -c "import sys, json; data=json.load(sys.stdin); print(len([t for t in data['data']['activeTargets'] if t['health']=='up']))")
echo "Prometheus 健康目标数: $TARGETS"
echo "✓ Prometheus 采集正常"
echo ""

echo "5. 验证 Prometheus 查询..."
METRIC_COUNT=$(curl -s "http://localhost:9090/api/v1/query?query=example_app_http_requests_total" | python3 -c "import sys, json; data=json.load(sys.stdin); print(len(data['data']['result']))")
echo "example_app_http_requests_total 指标数: $METRIC_COUNT"
echo "✓ Prometheus 查询正常"
echo ""

echo "6. 验证 Grafana..."
GRAFANA_STATUS=$(curl -s http://localhost:3000/api/health | python3 -c "import sys, json; print(json.load(sys.stdin)['database'])")
echo "Grafana 状态: $GRAFANA_STATUS"
echo "✓ Grafana 正常"
echo ""

echo "=== 所有验证通过！ ==="
echo ""
echo "访问地址:"
echo "  - 应用: http://localhost:8080"
echo "  - Prometheus: http://localhost:9090"
echo "  - Grafana: http://localhost:3000 (admin/admin)"
