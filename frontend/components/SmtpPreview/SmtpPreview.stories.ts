import { Meta, Story } from "@storybook/vue3";
import { normalizeSMTPEvent } from "~/utils/normalize-event";
import smtpOrderShippedEventMock from '~/mocks/smtp-order.json';
import smtpWelcomeEventMock from '~/mocks/smtp-welcome.json';
import SmtpPreview from '~/components/SmtpPreview/SmtpPreview.vue';

export default {
  title: "SMTP/Components/SmtpPreview",
  component: SmtpPreview
} as Meta<typeof SmtpPreview>;

const Template: Story = (args) => ({
  components: { SmtpPreview },
  setup() {
    return {
      args,
    };
  },
  template: `<SmtpPreview v-bind="args" />`,
});

export const OrderShipped = Template.bind({});

OrderShipped.args = {
  event: normalizeSMTPEvent(smtpOrderShippedEventMock),
};

export const Welcome = Template.bind({});

Welcome.args = {
  event: normalizeSMTPEvent(smtpWelcomeEventMock),
};
