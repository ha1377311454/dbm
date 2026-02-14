<template>
  <el-dialog
    v-model="visible"
    title="SQL 导出 - 类型映射预览"
    width="800px"
    @close="visible = false"
  >
    <!-- 映射摘要 -->
    <div class="mapping-summary">
      <el-alert type="success" :closable="false">
        <template #title>
          <div>
            ✅ 成功映射 {{ mappingResult.summary.direct }} 个类型
          </div>
        </template>
      </el-alert>

      <el-alert v-if="mappingResult.summary.fallback > 0" type="warning" :closable="false">
        <template #title>
          <div>
            ⚠ 有 {{ mappingResult.summary.fallback }} 个类型使用了安全降级
          </div>
        </template>
      </el-alert>

      <el-alert v-if="mappingResult.summary.userChoice > 0" type="info" :closable="false">
        <template #title>
          <div>
            ℹ️ 有 {{ mappingResult.summary.userChoice }} 个类型需要您选择
          </div>
        </template>
      </el-alert>

      <el-alert v-if="mappingResult.summary.lossyCount > 0" type="warning" :closable="false">
        <template #title>
          <div>
            ⚠️ 有 {{ mappingResult.summary.lossyCount }} 个类型存在精度损失
          </div>
        </template>
      </el-alert>

      <!-- 类型总数 -->
      <div class="mapping-stats">
        <span>总类型数: {{ mappingResult.summary.total }}</span>
        <el-divider direction="vertical" />
      </div>
    </div>

    <!-- 需要用户选择的类型 -->
    <div v-if="hasUserChoices" class="user-choices">
      <h3>需要您选择的类型（{{ Object.keys(mappingResult.requiresUser).length }} 个）</h3>
      <p class="tip">请为以下类型选择目标数据库类型：</p>

      <div v-for="(rule, sourceType) in mappingResult.requiresUser" :key="sourceType" class="choice-item">
        <div class="choice-header">
          <span class="source-type">{{ sourceType }}</span>
          <el-icon><Document /></el-icon>
          <el-tooltip v-if="rule.note" :content="rule.note">
            <el-icon><QuestionFilled /></el-icon>
          </el-tooltip>
        </div>

        <el-select v-model="userChoices[sourceType]" placeholder="请选择目标类型">
          <el-option
            v-for="opt in rule.userOptions"
            :key="opt.value"
            :value="opt.value"
            :label="opt.label"
          >
            <div class="option-item">
              <span>{{ opt.label }}</span>
              <el-tag v-if="isLossy(opt.value)" type="warning" size="small">
                可能有损失
              </el-tag>
            </div>
          </el-option>
        </el-select>
      </div>
    </div>

    <!-- 转换警告信息 -->
    <div v-if="mappingResult.warnings.length > 0" class="warnings-section">
      <h4>⚠️ 转换警告</h4>
      <ul>
        <li v-for="warning in mappingResult.warnings" :key="warning">
          {{ warning }}
        </li>
      </ul>
    </div>

    <!-- 操作按钮 -->
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="handlePreview">重新预览</el-button>
      <el-button type="primary" @click="handleConfirm" :disabled="!canConfirm">
        确认并导出
        <el-icon><CircleCheck /></el-icon>
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Right, Document, QuestionFilled, CircleCheck } from '@element-plus/icons-vue'

const props = defineProps<{
  mappingResult: export.TypeMappingResult
}>()

const emit = defineEmits<{
  confirm: [choices: Map<string, string>]
  preview: []
}>()

const userChoices = ref<Map<string, string>>({})

const hasUserChoices = computed(() => {
  return Object.keys(props.mappingResult.requiresUser).length > 0
})

const canConfirm = computed(() => {
  // 检查所有需要用户选择的类型是否已选择
  for (const sourceType in Object.keys(props.mappingResult.requiresUser)) {
    if (!userChoices.value[sourceType]) {
      return false
    }
  }
  return true
})

const isLossy = (targetType: string) => {
  // 判断是否有精度损失
  // 这里可以根据类型名称判断，也可以在 TypeRule 中添加标记
  const lossyTypes = ['TINYINT', 'SMALLINT', 'TINYINT_UNSIGNED', 'FLOAT']
  return lossyTypes.some(type => targetType.includes(type))
}

const handlePreview = () => {
  emit('preview')
}

const handleConfirm = () => {
  // 应用用户选择的类型
  emit('confirm', userChoices.value)
}
</script>

<style scoped>
.mapping-summary {
  margin-bottom: 20px;
  padding: 15px;
}

.mapping-stats {
  display: flex;
  align-items: center;
  gap: 15px;
  font-size: 13px;
  color: #909399;
}

.choice-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 15px;
  background: #f5f7fa;
  border-radius: 4px;
}

.choice-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.source-type {
  font-family: 'Courier New', monospace;
  background: #e6f7ff;
  padding: 4px 8px;
  border-radius: 3px;
  font-weight: 500;
}

.warnings-section {
  margin-bottom: 20px;
  padding: 15px;
  background: #fff3cd;
  border-radius: 4px;
}

.warnings h4 {
  margin: 0 0 10px 0;
  color: #e6a23c;
}

.warnings ul {
  margin: 0;
  padding-left: 20px;
}

.user-choices {
  margin-bottom: 20px;
}

.choice-item {
  margin-bottom: 15px;
}
</style>
