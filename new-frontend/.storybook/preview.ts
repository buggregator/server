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
};


const pinia = createPinia();

app.use(pinia);
