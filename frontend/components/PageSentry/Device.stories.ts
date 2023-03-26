import { Meta, Story } from "@storybook/vue3";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import sentryEventMock from '~/mocks/sentry-common.json'
import DeviceComponent from './Device.vue';

export default {
  title: "Pages/Sentry/Parts",
  component: DeviceComponent
} as Meta<typeof DeviceComponent>;

const Template: Story = (args) => ({
  components: { DeviceComponent },
  setup() {
    return {
      args,
    };
  },
  template: `<DeviceComponent v-bind="args" />`,
});

export const Device = Template.bind({});

Device.args = {
  event: normalizeSentryEvent(sentryEventMock).payload,
};