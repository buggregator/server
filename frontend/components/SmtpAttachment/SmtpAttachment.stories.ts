import {Meta, Story} from "@storybook/vue3";
import SmtpAttachment from '~/components/SmtpAttachment/SmtpAttachment.vue';

export default {
  title: "Components/SmtpAttachment",
  component: SmtpAttachment
} as Meta<typeof SmtpAttachment>;

const Template: Story = (args) => ({
  components: {SmtpAttachment},
  setup() {
    return {
      args,
    };
  },
  template: `
    <SmtpAttachment v-bind="args"/>`,
});

export const Default = Template.bind({});

Default.args = {
  event: {id: 'cbdd3296-1e25-4191-9f52-0e2d7e7d6aae'},
  attachment: {
    id: 'cbdd3296-1e25-4191-9f52-0e2d7e7d6aae',
    name: 'attachment.txt',
    size: 234234,
    mime: "text/plain",
    uri: 'example.com/attachment.txt',
  }
};
