/* Tailwind CSS Custom Configuration for SidBpfKubers */
tailwind.config = {
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        obsidian: '#0D1B1A', // Bottle Green / Deep Forest canvas
        surface: {
          DEFAULT: '#112220', // Archival deep forest surface
          alt: '#162B28',     // Muted bottle green elevated
          border: '#1F3834',  // Analog deep forest border
          hover: '#1A3330'
        },
        cyber: {
          mint: '#00F5A0',
          emerald: '#10B981',
          jade: '#34D399',
          blue: '#10B981', // Mapped to emerald for unified deep forest + green theme
          amber: '#FFB300',
          purple: '#BF5AF2',
          crimson: '#FF3B30'
        }
      },
      fontFamily: {
        sans: ['Inter', '-apple-system', 'BlinkMacSystemFont', 'sans-serif'],
        mono: ['JetBrains Mono', 'SF Pro Mono', 'monospace']
      }
    }
  }
};
