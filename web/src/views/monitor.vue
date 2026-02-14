<template>
  <div class="monitor-page">
    <el-page-header title="系统监控" />

    <el-row :gutter="20" style="margin-top: 20px">
      <el-col :span="6">
        <el-card shadow="hover">
          <el-statistic title="活跃连接" :value="stats.connections">
            <template #prefix>
              <el-icon><ConnectionIcon /></el-icon>
            </template>
          </el-statistic>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <el-statistic title="查询总数" :value="stats.queries">
            <template #prefix>
              <el-icon><DataAnalysis /></el-icon>
            </template>
          </el-statistic>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <el-statistic title="平均响应时间" :value="stats.avgTime" suffix="ms">
            <template #prefix>
              <el-icon><Timer /></el-icon>
            </template>
          </el-statistic>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <el-statistic title="慢查询数" :value="stats.slowQueries">
            <template #prefix>
              <el-icon><Warning /></el-icon>
            </template>
          </el-statistic>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px">
      <el-col :span="12">
        <el-card header="查询趋势">
          <div ref="queryTrendChart" style="height: 300px"></div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card header="响应时间分布">
          <div ref="responseTimeChart" style="height: 300px"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-row style="margin-top: 20px">
      <el-col :span="24">
        <el-card header="Prometheus 指标">
          <el-text tag="pre" style="white-space: pre-wrap">{{ prometheusMetrics }}</el-text>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import * as echarts from 'echarts'
import { Connection as ConnectionIcon, DataAnalysis, Timer, Warning } from '@element-plus/icons-vue'

const stats = ref({
  connections: 0,
  queries: 0,
  avgTime: 0,
  slowQueries: 0
})

const prometheusMetrics = ref(`# Prometheus 指标端点: /metrics
# dbm_connections_active 活跃连接数
# dbm_queries_total 查询总数
# dbm_query_duration_seconds 查询耗时
# dbm_slow_queries_total 慢查询总数`)

const queryTrendChart = ref<HTMLElement>()
const responseTimeChart = ref<HTMLElement>()

let queryTrend: echarts.ECharts | null = null
let responseTime: echarts.ECharts | null = null

onMounted(() => {
  initCharts()
  loadStats()
})

onBeforeUnmount(() => {
  queryTrend?.dispose()
  responseTime?.dispose()
})

function initCharts() {
  if (queryTrendChart.value) {
    queryTrend = echarts.init(queryTrendChart.value)
    queryTrend.setOption({
      xAxis: { type: 'category', data: ['00:00', '04:00', '08:00', '12:00', '16:00', '20:00'] },
      yAxis: { type: 'value' },
      series: [{
        data: [120, 200, 150, 80, 70, 110],
        type: 'line',
        smooth: true,
        areaStyle: { opacity: 0.3 }
      }]
    })
  }

  if (responseTimeChart.value) {
    responseTime = echarts.init(responseTimeChart.value)
    responseTime.setOption({
      xAxis: { type: 'category', data: ['<100ms', '100-500ms', '500ms-1s', '1s-3s', '>3s'] },
      yAxis: { type: 'value' },
      series: [{
        data: [800, 120, 50, 20, 10],
        type: 'bar',
        itemStyle: { color: '#409eff' }
      }]
    })
  }
}

async function loadStats() {
  // 模拟数据
  stats.value = {
    connections: 5,
    queries: 1234,
    avgTime: 45,
    slowQueries: 8
  }
}
</script>

<style scoped>
.monitor-page {
  padding: 20px;
}

:deep(.el-statistic__head) {
  font-size: 14px;
  color: #909399;
}

:deep(.el-statistic__content) {
  font-size: 28px;
  font-weight: bold;
}
</style>
