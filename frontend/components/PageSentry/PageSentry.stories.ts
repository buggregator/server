import { Meta, Story } from "@storybook/vue3";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import sentryEventMock from '~/mocks/sentry-common.json'
import PageSentry from './PageSentry.vue';

export default {
  title: "Pages/Sentry",
  component: PageSentry
} as Meta<typeof PageSentry>;

const Template: Story = (args) => ({
  components: { PageSentry },
  setup() {
    return {
      args,
    };
  },
  template: `<page-sentry v-bind="args" />`,
});

export const Page = Template.bind({});

Page.args = {
  event: normalizeSentryEvent(sentryEventMock),
};
