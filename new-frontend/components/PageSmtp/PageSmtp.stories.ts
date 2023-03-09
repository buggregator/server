import { Meta, Story } from "@storybook/vue3";
import { normalizeSMTPEvent } from "~/utils/normalize-event";
import smtpEventMock from '~/mocks/smtp-welcome.json'
import PageSmtp from "./PageSmtp.vue";

export default {
  title: "Pages/Smtp",
  component: PageSmtp
} as Meta<typeof PageSmtp>;

const Template: Story = (args) => ({
  components: { PageSmtp },
  setup() {
    return {
      args,
    };
  },
  template: `<page-smtp v-bind="args" />`,
});

export const Default = Template.bind({});

const normalizeEvent = normalizeSMTPEvent(smtpEventMock)

Default.args = {
  event: normalizeEvent,
  HTML: normalizeEvent.payload.html
};
