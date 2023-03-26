import { Meta, Story } from "@storybook/vue3";
import { normalizeSMTPEvent } from "~/utils/normalize-event";
import EventSmtp from '~/components/EventSmtp/EventSmtp.vue';
import smtpOrderShippedEventMock from '~/mocks/smtp-order.json';
import smtpWelcomeEventMock from '~/mocks/smtp-welcome.json';

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

export const OrderShipped = Template.bind({});

OrderShipped.args = {
  event: normalizeSMTPEvent(smtpOrderShippedEventMock),
};

export const Welcome = Template.bind({});

Welcome.args = {
  event: normalizeSMTPEvent(smtpWelcomeEventMock),
};
