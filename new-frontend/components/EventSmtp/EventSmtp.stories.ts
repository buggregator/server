import { normalizeSMTPEvent } from "~/utils/normalizeEvent";
import { Meta, Story } from "@storybook/vue3";
import EventSmtp from './EventSmtp.vue';
import smtpEventMock from '../../mocks/smtp.json'

export default {
  title: "Event/EventSMTP",
  component: EventSmtp
} as Meta<typeof EventSmtp>;

const Template: Story = (args) => ({
  components: { EventSmtp },
  setup() {
    return {
      args,
    };
  },
  template: `<event-smtp v-bind="args" />`,
});

export const Default = Template.bind({});

Default.args = {
  event: normalizeSMTPEvent(smtpEventMock),
};
