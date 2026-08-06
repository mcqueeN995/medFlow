#!/usr/bin/env node
// Парсит JSON-репортёры Vitest/Playwright и пушит агрегаты в Prometheus
// Pushgateway - единственный штатный способ увидеть результаты разовых
// тест-прогонов в Grafana (Prometheus сам умеет только scrape, см.
// infra/prometheus/prometheus.yml и README).
import { readFileSync } from 'node:fs'
import path from 'node:path'

const pushgatewayUrl = process.env.PUSHGATEWAY_URL ?? `http://localhost:${process.env.PUSHGATEWAY_PORT ?? 9091}`
const root = path.join(import.meta.dirname, '..')

function readJson(file) {
  try {
    return JSON.parse(readFileSync(path.join(root, file), 'utf8'))
  } catch {
    return null
  }
}

function vitestSummary() {
  const report = readJson('vitest-results.json')
  if (!report) return null
  return {
    passed: report.numPassedTests ?? 0,
    failed: report.numFailedTests ?? 0,
    total: report.numTotalTests ?? 0,
  }
}

function playwrightSummary() {
  const report = readJson('e2e-results.json')
  if (!report) return null
  const stats = report.stats ?? {}
  const passed = stats.expected ?? 0
  const failed = (stats.unexpected ?? 0) + (stats.flaky ?? 0)
  const skipped = stats.skipped ?? 0
  return { passed, failed, total: passed + failed + skipped }
}

async function pushSuite(suite, summary) {
  if (!summary) {
    console.log(`[push-test-metrics] no report found for suite=${suite}, skipping`)
    return
  }
  const now = Math.floor(Date.now() / 1000)
  const body = [
    `# TYPE medflow_test_passed gauge`,
    `medflow_test_passed{suite="${suite}"} ${summary.passed}`,
    `# TYPE medflow_test_failed gauge`,
    `medflow_test_failed{suite="${suite}"} ${summary.failed}`,
    `# TYPE medflow_test_total gauge`,
    `medflow_test_total{suite="${suite}"} ${summary.total}`,
    `# TYPE medflow_test_last_run_timestamp gauge`,
    `medflow_test_last_run_timestamp{suite="${suite}"} ${now}`,
    '',
  ].join('\n')

  const url = `${pushgatewayUrl}/metrics/job/medflow_frontend_tests/suite/${suite}`
  const res = await fetch(url, { method: 'POST', body })
  if (!res.ok) {
    throw new Error(`push to pushgateway failed for suite=${suite}: ${res.status} ${await res.text()}`)
  }
  console.log(`[push-test-metrics] suite=${suite} passed=${summary.passed} failed=${summary.failed} total=${summary.total}`)
}

await pushSuite('vitest', vitestSummary())
await pushSuite('playwright', playwrightSummary())
