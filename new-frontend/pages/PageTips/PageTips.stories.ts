import { Meta, Story } from "@storybook/vue3";
import PageTips from "~/pages/PageTips/PageTips.vue";

export default {
  title: "Pages/PageTips",
  component: PageTips
}as Meta<typeof PageTips>;

const Template: Story = (args) => ({
  components: { PageTips },
  setup() {
    return {
      args,
    };
  },
  template: '<page-tips v-bind="args" />',
});

export const Default = Template.bind({});
Default.args = {
};
