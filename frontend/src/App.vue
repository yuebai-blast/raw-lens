<script setup lang="ts">
import Masthead from '@/components/Masthead.vue'
// 轮询生命周期已移交给 InspectorView（真正展示抓包列表的视图），
// 避免绑死在只挂载一次的 App.vue 上导致登录后不自动刷新。
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
</template>
