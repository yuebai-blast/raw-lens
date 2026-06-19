<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useCaptureStore } from '@/stores/captures'
import Masthead from '@/components/Masthead.vue'

const store = useCaptureStore()
onMounted(() => store.startPolling())
onUnmounted(() => store.stopPolling())
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
