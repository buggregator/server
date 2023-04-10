import { Meta, Story } from "@storybook/vue3";
import PreviewCardFooter from "~/components/PreviewCardFooter/PreviewCardFooter.vue";

export default {
  title: "Preview/PreviewCardFooter",
  component: PreviewCardFooter,
}as Meta<typeof PreviewCardFooter>;

const Template: Story = (args: typeof Object) => ({
  components: { PreviewCardFooter },
  setup() {
    return {
      args,
    };
  },
  template: '<PreviewCardFooter v-bind="args" />',
});

export const Default = Template.bind({});
Default.args = {
  serverName: "My server",
  originConfig: {
    one: 1,
    two: 2,
  },
};
