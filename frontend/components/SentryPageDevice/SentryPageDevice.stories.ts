import { Meta, Story } from "@storybook/vue3";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import sentryEventMock from '~/mocks/sentry-common.json'
import SentryPageDevice from '~/components/SentryPageDevice/SentryPageDevice.vue';

export default {
  title: "Sentry/Page/SentryPageDevice",
  component: SentryPageDevice
} as Meta<typeof SentryPageDevice>;

const Template: Story = (args) => ({
  components: { SentryPageDevice },
  setup() {
    return {
      args,
    };
  },
  template: `<SentryPageDevice v-bind="args" />`,
});

export const Device = Template.bind({});

Device.args = {
  event: normalizeSentryEvent(sentryEventMock).payload,
};
