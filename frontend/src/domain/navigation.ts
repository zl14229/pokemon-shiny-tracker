import { onBeforeUnmount, onMounted, ref } from 'vue'
export function useRoute() {
  const route = ref(location.hash.slice(1) || '/home')
  const update = () => {
    route.value = location.hash.slice(1) || '/home'
  }
  onMounted(() => window.addEventListener('hashchange', update))
  onBeforeUnmount(() => window.removeEventListener('hashchange', update))
  return route
}
export function go(path: string) {
  location.hash = path
}
