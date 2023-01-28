import autoprefixer from 'autoprefixer'
import tailwind from 'tailwindcss'
import tailwindConfig from './tailwind.config.js'

export default {
  plugins: [autoprefixer, tailwind(tailwindConfig)]
};
