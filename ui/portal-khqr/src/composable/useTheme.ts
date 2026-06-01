import {ref, watch} from 'vue'
import type {ThemeMode} from '../types'

const theme = ref<ThemeMode>('light')
export function useTheme() {
    const toggleTheme = () => {
        theme.value = theme.value === 'light' ? 'dark' : 'light'
    }

    watch(theme, 
        (val) => {
            document.documentElement.setAttribute('data-theme', val)
        },
        { immediate: true }
    )
    return { theme, toggleTheme }
}