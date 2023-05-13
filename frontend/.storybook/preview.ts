import {createPinia} from 'pinia';
import {addParameters, app} from '@storybook/vue3';
import "../assets/index.css";
import "../assets/index";
import SfdumpWrap from "../vendor/dumper";

addParameters({
  actions: {argTypesRegex: "^on[A-Z].*"},
  controls: {
    matchers: {
      color: /(background|color)$/i,
      date: /Date$/,
    },
  },
  themes: {
    clearable: false,
    target: 'html',
    default: 'dark',
    list: [
      {
        name: 'light',
        class: [],
        color: '#ffffff',
      },
      {
        name: 'dark',
        class: ['dark'],
        color: '#333333'
      }
    ]
  }
})


const pinia = createPinia();

app.use(pinia);

declare global {
  interface Window {
    Sfdump: (id: string) => void;
  }
}
window.Sfdump = SfdumpWrap(window.document)
