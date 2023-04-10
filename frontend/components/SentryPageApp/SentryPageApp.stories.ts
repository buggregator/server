import { Meta, Story } from "@storybook/vue3";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import sentryEventMock from '~/mocks/sentry-common.json'
import SentryPageApp from '~/components/SentryPageApp/SentryPageApp.vue';

export default {
  title: "Sentry/Page/SentryPageApp",
  component: SentryPageApp
} as Meta<typeof SentryPageApp>;

const Template: Story = (args) => ({
  components: { SentryPageApp },
  setup() {
    return {
      args,
    };
  },
  template: `<SentryPageApp v-bind="args" />`,
});

export const App = Template.bind({});

App.args = {
  event: normalizeSentryEvent(sentryEventMock).payload,
};
