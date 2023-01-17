import { Meta, Story } from "@storybook/vue3";
import { normalizeSMTPEvent } from "~/utils/normalize-event";
import smtpEventMock from '~/mocks/smtp-welcome.json'
import SmtpPage from "~/components/SmtpPage/SmtpPage.vue";

export default {
  title: "Smtp/Page/SmtpPage",
  component: SmtpPage
} as Meta<typeof SmtpPage>;

const Template: Story = (args) => ({
  components: { SmtpPage },
  setup() {
    return {
      args,
    };
  },
  template: `<SmtpPage v-bind="args" />`,
});

export const Default = Template.bind({});

const normalizeEvent = normalizeSMTPEvent(smtpEventMock)

Default.args = {
  event: normalizeEvent,
  htmlSource: normalizeEvent.payload.html
};
