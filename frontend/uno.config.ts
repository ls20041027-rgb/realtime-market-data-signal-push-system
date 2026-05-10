import {
  defineConfig,
  presetUno,
  presetAttributify,
  presetIcons,
  transformerVariantGroup,
} from 'unocss'

// UnoCSS 配置：与 src/styles/vars.css 的 CSS 变量打通
export default defineConfig({
  presets: [presetUno(), presetAttributify(), presetIcons()],
  transformers: [transformerVariantGroup()],
  theme: {
    colors: {
      bg: 'var(--bg)',
      panel: 'var(--panel)',
      divider: 'var(--divider)',
      text: {
        DEFAULT: 'var(--text)',
        dim: 'var(--text-dim)',
      },
      up: 'var(--up)',
      down: 'var(--down)',
      neutral: 'var(--neutral)',
      warn: 'var(--warn)',
      info: 'var(--info)',
      critical: 'var(--critical)',
    },
    fontFamily: {
      mono: 'var(--font-mono)',
      sans: 'var(--font-sans)',
    },
  },
  shortcuts: {
    'panel-card': 'bg-panel border border-divider rounded-md',
    'text-up': 'color-up',
    'text-down': 'color-down',
    'text-neutral': 'color-neutral',
  },
})
