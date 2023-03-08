import { Meta, Story } from "@storybook/vue3";
import { normalizeSMTPEvent } from "~/utils/normalize-event";
import EventSmtp from '~/components/EventSmtp/EventSmtp.vue';
import smtpEventMock from '~/mocks/smtp.json'

export default {
  title: "SMTP",
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

export const Event = Template.bind({});

Event.args = {
  event: normalizeSMTPEvent(smtpEventMock),
};
