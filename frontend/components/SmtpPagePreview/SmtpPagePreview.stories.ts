import { Meta, Story } from "@storybook/vue3";
import { HTML } from '@/mocks/mail';
import SmtpPagePreview from '~/components/SmtpPagePreview/SmtpPagePreview.vue';

export default {
  title: "Smtp/Page/SmtpPagePreview",
  component: SmtpPagePreview
} as Meta<typeof SmtpPagePreview>;

const Template: Story = (args) => ({
  components: { SmtpPagePreview },
  setup() {
    return {
      args,
    };
  },
  template: `<SmtpPagePreview v-bind="args">${HTML}</SmtpPagePreview>`,
});

export const Tablet = Template.bind({});

Tablet.args = {
  device: 'tablet',
};

export const Mobile = Template.bind({});

Mobile.args = {
  device: 'mobile',
};

export const Desktop = Template.bind({});

Desktop.args = {
  device: 'desktop',
};
