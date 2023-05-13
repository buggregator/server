import {Meta, Story} from "@storybook/vue3";
import Attachment from '~/components/Attachment/Attachment.vue';

export default {
  title: "Components/Attachment",
  component: Attachment
} as Meta<typeof Attachment>;

const Template: Story = (args) => ({
  components: {Attachment},
  setup() {
    return {
      args,
    };
  },
  template: `
    <Attachment v-bind="args"/>`,
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
