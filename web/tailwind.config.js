/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        // Terminal/infra aesthetic: monospace everywhere.
        sans: [
          'JetBrains Mono',
          'IBM Plex Mono',
          'ui-monospace',
          'SFMono-Regular',
          'Menlo',
          'Consolas',
          'monospace'
        ],
        mono: [
          'JetBrains Mono',
          'IBM Plex Mono',
          'ui-monospace',
          'SFMono-Regular',
          'Menlo',
          'Consolas',
          'monospace'
        ]
      },
      colors: {
        // Repointed from indigo to terminal green — every `brand-*` usage
        // across the app flips to green automatically.
        brand: {
          50: '#eafff5',
          100: '#c9ffe6',
          200: '#93fbcd',
          300: '#4df3aa',
          400: '#1fe087',
          500: '#12c06d',
          600: '#0e9a58',
          700: '#0d7a47',
          800: '#0f603a',
          900: '#0f4e31',
          950: '#042c1c'
        },
        // Near-black terminal surfaces with a faint green tint.
        ink: {
          950: '#05080a',
          900: '#0a0f0d',
          850: '#0e1512',
          800: '#131b17',
          700: '#1a241f'
        }
      },
      borderRadius: {
        // Sharper corners read as "terminal".
        lg: '0.375rem',
        xl: '0.5rem'
      },
      boxShadow: {
        soft: '0 1px 2px 0 rgba(0, 0, 0, 0.4)',
        glow: '0 0 0 1px rgba(31,224,135,0.25), 0 0 20px -6px rgba(31,224,135,0.45)',
        term: 'inset 0 0 0 1px rgba(31,224,135,0.08)'
      },
      backgroundImage: {
        // Faint grid backdrop for the app shell.
        grid: 'linear-gradient(rgba(31,224,135,0.035) 1px, transparent 1px), linear-gradient(90deg, rgba(31,224,135,0.035) 1px, transparent 1px)',
        scanline:
          'repeating-linear-gradient(0deg, rgba(255,255,255,0.015) 0px, rgba(255,255,255,0.015) 1px, transparent 1px, transparent 3px)'
      },
      backgroundSize: {
        grid: '32px 32px'
      },
      keyframes: {
        'fade-in': {
          '0%': { opacity: '0', transform: 'translateY(4px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        shimmer: {
          '0%': { backgroundPosition: '-400px 0' },
          '100%': { backgroundPosition: '400px 0' }
        },
        blink: {
          '0%, 49%': { opacity: '1' },
          '50%, 100%': { opacity: '0' }
        }
      },
      animation: {
        'fade-in': 'fade-in 150ms ease-out',
        shimmer: 'shimmer 1.4s linear infinite',
        blink: 'blink 1.1s steps(1) infinite'
      }
    }
  },
  plugins: []
}
