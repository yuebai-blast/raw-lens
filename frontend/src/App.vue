<script setup lang="ts">
import Masthead from '@/components/Masthead.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import { useConfirmStore } from '@/stores/confirm'
// 轮询生命周期已移交给 InspectorView（真正展示抓包列表的视图），
// 避免绑死在只挂载一次的 App.vue 上导致登录后不自动刷新。
const confirm = useConfirmStore()
</script>

<template>
  <div
    class="fx-scanlines"
    aria-hidden="true"
  />
  <div
    class="fx-vignette"
    aria-hidden="true"
  />
  <Masthead />
  <router-view />
  <ConfirmDialog
    :open="confirm.open"
    :title="confirm.title"
    :message="confirm.message"
    :confirm-text="confirm.confirmText"
    :cancel-text="confirm.cancelText"
    @confirm="confirm.accept()"
    @cancel="confirm.cancel()"
  />
</template>
