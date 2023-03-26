import { Meta, Story } from "@storybook/vue3";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import sentryEventMock from '~/mocks/sentry-common.json'
import { Sentry } from "~/config/types";
import SentryException from './SentryException.vue';

export default {
  title: "Sentry/Components",
  component: SentryException
} as Meta<typeof SentryException>;

const Template: Story = (args) => ({
  components: { SentryException },
  setup() {
    return {
      args,
    };
  },
  template: `<SentryException v-bind="args" />`,
});

export const Exception = Template.bind({});

Exception.args = {
  exception: (normalizeSentryEvent(sentryEventMock)?.payload as Sentry)?.exception?.values[0],
};