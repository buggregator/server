import { Meta, Story } from "@storybook/vue3";
import LayoutTips from "~/components/LayoutTips/LayoutTips.vue";

export default {
  title: "Layouts/LayoutTips",
  component: LayoutTips
}as Meta<typeof LayoutTips>;

const Template: Story = (args) => ({
  components: { LayoutTips },
  setup() {
    return {
      args,
    };
  },
  template: '<layout-tips v-bind="args" />',
});

export const Default = Template.bind({});
Default.args = {
};
