import { ref } from 'vue'

export function useCopy() {
  const copied = ref(false)
  function copy(text: string) {
    void navigator.clipboard.writeText(text).then(() => {
      copied.value = true
      setTimeout(() => (copied.value = false), 1200)
    })
  }
  return { copied, copy }
}
