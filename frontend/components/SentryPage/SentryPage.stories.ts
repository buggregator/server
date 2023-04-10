import { Meta, Story } from "@storybook/vue3";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import sentryEventMock from '~/mocks/sentry-common.json'
import SentryPage from '~/components/SentryPage/SentryPage.vue';

export default {
  title: "Sentry/Page/SentryPage",
  component: SentryPage
} as Meta<typeof SentryPage>;

const Template: Story = (args) => ({
  components: { SentryPage },
  setup() {
    return {
      args,
    };
  },
  template: `<SentryPage v-bind="args" />`,
});

export const Page = Template.bind({});

Page.args = {
  event: normalizeSentryEvent(sentryEventMock),
};
