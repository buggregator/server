import { Meta, Story } from "@storybook/vue3";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import sentryEventMock from '~/mocks/sentry-common.json'
import AppComponent from './App.vue';

export default {
  title: "Pages/Sentry/Parts",
  component: AppComponent
} as Meta<typeof AppComponent>;

const Template: Story = (args) => ({
  components: { AppComponent },
  setup() {
    return {
      args,
    };
  },
  template: `<AppComponent v-bind="args" />`,
});

export const App = Template.bind({});

App.args = {
  event: normalizeSentryEvent(sentryEventMock).payload,
};