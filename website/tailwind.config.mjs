/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}'],
  theme: {
    extend: {
      colors: {
        // Warm cream background
        cream: '#faf8f5',
        // Near-black body text
        ink: '#1c1c1c',
        // Warm gray muted text
        muted: '#6b6561',
        // Terminal colors
        terminal: {
          bg: '#0f1115',
          text: '#e6e4e0',
        },
        // Primary accent - Instrument teal
        teal: {
          DEFAULT: '#1d7a74',
          dark: '#165f5a',
          light: '#2a9a93',
        },
        // Callout/warning - Muted red
        warning: '#b84c3f',
      },
      fontFamily: {
        sans: ['Jost', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Menlo', 'Monaco', 'monospace'],
      },
      fontSize: {
        // Slightly larger base for readability
        base: ['1.0625rem', '1.6'],
        lg: ['1.1875rem', '1.6'],
        xl: ['1.375rem', '1.5'],
        '2xl': ['1.625rem', '1.4'],
        '3xl': ['2rem', '1.3'],
        '4xl': ['2.5rem', '1.2'],
        '5xl': ['3.25rem', '1.1'],
      },
      borderWidth: {
        hairline: '1px',
      },
      boxShadow: {
        'code': '0 4px 16px rgba(0, 0, 0, 0.12)',
        'card': '0 2px 8px rgba(0, 0, 0, 0.06)',
      },
      spacing: {
        '18': '4.5rem',
        '22': '5.5rem',
      },
    },
  },
  plugins: [],
};
