import { createPinia } from 'pinia';
import { app } from '@storybook/vue3';
import "../assets/index.css";
import "../assets/index"

export const parameters = {
  actions: { argTypesRegex: "^on[A-Z].*" },
  controls: {
    matchers: {
      color: /(background|color)$/i,
      date: /Date$/,
    },
  },
  themes: {
    clearable: false,
    target: 'html',
    list: [
      {
        name: 'Light',
        class: [],
        color: '#ffffff',
        default: true,
      },
      {
        name: 'Dark',
        class: ['dark'],
        color: '#000000'
      }
    ]
  }
};

const pinia = createPinia();

app.use(pinia);
