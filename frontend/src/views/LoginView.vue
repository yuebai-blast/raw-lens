<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const username = ref('')
const password = ref('')
const error = ref('')
const busy = ref(false)

async function submit() {
  error.value = ''
  busy.value = true
  const ok = await auth.login(username.value, password.value)
  busy.value = false
  if (ok) {
    const redirect = (route.query.redirect as string) || '/'
    void router.replace(redirect)
  } else {
    error.value = '用户名或密码错误'
  }
}
</script>

<template>
  <div class="login-wrap">
    <form
      class="login-card"
      @submit.prevent="submit"
    >
      <h1>raw<span class="sep">·</span>lens</h1>
      <p class="sub">
        WIRE-LEVEL HTTP INSPECTOR
      </p>
      <label>
        <span>USERNAME</span>
        <input
          v-model="username"
          autocomplete="username"
          autofocus
          required
        >
      </label>
      <label>
        <span>PASSWORD</span>
        <input
          v-model="password"
          type="password"
          autocomplete="current-password"
          required
        >
      </label>
      <p
        v-if="error"
        class="err"
      >
        {{ error }}
      </p>
      <button
        type="submit"
        :disabled="busy"
      >
        {{ busy ? '...' : 'SIGN IN' }}
      </button>
    </form>
  </div>
</template>

<style scoped>
.login-wrap {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
}
.login-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
  width: 320px;
  padding: 28px 26px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: linear-gradient(180deg, var(--panel-2), var(--panel));
}
.login-card h1 {
  margin: 0;
  font-family: var(--mono);
  font-weight: 500;
  font-size: 22px;
  letter-spacing: 1px;
  color: var(--ink);
  text-shadow: 0 0 14px #34e0a140;
}
.login-card h1 .sep {
  color: var(--phosphor);
}
.sub {
  margin: 0 0 6px;
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 3.5px;
  color: var(--muted);
}
.login-card label {
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.login-card label span {
  font-family: var(--mono);
  font-size: 9px;
  letter-spacing: 2px;
  color: var(--muted);
}
.login-card input {
  font-family: var(--mono);
  font-size: 13px;
  color: var(--ink);
  background: #0a0f0e;
  border: 1px solid var(--line);
  border-radius: 5px;
  padding: 8px 10px;
  outline: none;
}
.login-card input:focus {
  border-color: var(--phosphor);
  box-shadow: 0 0 10px #34e0a122;
}
.err {
  margin: 0;
  font-family: var(--mono);
  font-size: 11px;
  color: var(--red);
}
.login-card button {
  font-family: var(--mono);
  font-size: 12px;
  letter-spacing: 2px;
  font-weight: 500;
  color: var(--ink);
  background: #0a0f0e;
  border: 1px solid var(--line);
  border-radius: 5px;
  padding: 9px 0;
  cursor: pointer;
  transition: all 0.15s;
}
.login-card button:hover:not(:disabled) {
  border-color: var(--phosphor);
  box-shadow: 0 0 12px #34e0a122;
}
.login-card button:disabled {
  opacity: 0.5;
  cursor: default;
}
</style>
